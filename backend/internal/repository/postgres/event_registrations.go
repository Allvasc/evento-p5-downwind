package postgres

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Inscrições pelo formulário da página do evento (modo "form"): sem conta e sem
// pagamento. Cada evento tem as suas — a lista, o CSV e as métricas nunca misturam
// eventos.

var ErrRegistrationDuplicate = errors.New("este e-mail já está inscrito neste evento")
var ErrRegistrationFull = errors.New("as vagas deste evento estão esgotadas")
var ErrRegistrationClosed = errors.New("este evento não está recebendo inscrições pelo formulário")

const (
	RegistrationConfirmed = "confirmed"
	RegistrationWaitlist  = "waitlist"
	RegistrationCancelled = "cancelled"
)

type EventRegistrationRepository struct {
	pool *pgxpool.Pool
}

func NewEventRegistrationRepository(pool *pgxpool.Pool) *EventRegistrationRepository {
	return &EventRegistrationRepository{pool: pool}
}

type NewRegistration struct {
	FullName      string
	WhatsApp      string
	Email         string
	PracticeLevel string
	Interest      string
	WeightKg      *int
	ConsentText   string
}

type RegistrationResult struct {
	ID     string
	Status string // confirmed | waitlist
	// WaitlistPosition is 1-based, only set when Status is waitlist.
	WaitlistPosition int
}

// Register stores one sign-up. The event row is locked for the whole transaction, so
// two people racing for the last spot can't both be confirmed: the loser lands on the
// waitlist (or gets ErrRegistrationFull when the event has no waitlist).
func (r *EventRegistrationRepository) Register(ctx context.Context, eventID string, in NewRegistration) (*RegistrationResult, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var mode string
	var capacity *int
	var waitlist, active bool
	err = tx.QueryRow(ctx, `
		SELECT registration_mode, form_capacity, waitlist_enabled, active FROM events WHERE id = $1 FOR UPDATE
	`, eventID).Scan(&mode, &capacity, &waitlist, &active)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if !active || mode != RegistrationForm {
		return nil, ErrRegistrationClosed
	}

	status := RegistrationConfirmed
	if capacity != nil {
		var confirmed int
		if err := tx.QueryRow(ctx, `
			SELECT COUNT(*) FROM event_registrations WHERE event_id = $1 AND status = 'confirmed'
		`, eventID).Scan(&confirmed); err != nil {
			return nil, err
		}
		if confirmed >= *capacity {
			if !waitlist {
				return nil, ErrRegistrationFull
			}
			status = RegistrationWaitlist
		}
	}

	now := time.Now()
	var id string
	err = tx.QueryRow(ctx, `
		INSERT INTO event_registrations (event_id, full_name, whatsapp, email, practice_level, interest, weight_kg, status,
		                                 rules_accepted_at, consent_at, consent_text)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $9, $10)
		RETURNING id
	`, eventID, in.FullName, in.WhatsApp, in.Email, in.PracticeLevel, in.Interest, in.WeightKg, status, now, in.ConsentText).Scan(&id)
	if isUniqueViolation(err) {
		return nil, ErrRegistrationDuplicate
	}
	if err != nil {
		return nil, err
	}

	result := &RegistrationResult{ID: id, Status: status}
	if status == RegistrationWaitlist {
		if err := tx.QueryRow(ctx, `
			SELECT COUNT(*) FROM event_registrations
			WHERE event_id = $1 AND status = 'waitlist' AND created_at <= (SELECT created_at FROM event_registrations WHERE id = $2)
		`, eventID, id).Scan(&result.WaitlistPosition); err != nil {
			return nil, err
		}
	}
	return result, tx.Commit(ctx)
}

// Spots is what the public page shows: the limit (nil = sem limite) and how many are
// already confirmed.
func (r *EventRegistrationRepository) Spots(ctx context.Context, eventID string) (capacity *int, confirmed int, err error) {
	err = r.pool.QueryRow(ctx, `
		SELECT e.form_capacity,
		       (SELECT COUNT(*) FROM event_registrations er WHERE er.event_id = e.id AND er.status = 'confirmed')
		FROM events e WHERE e.id = $1
	`, eventID).Scan(&capacity, &confirmed)
	return capacity, confirmed, err
}

// RecordView counts one page view for today (Fortaleza). Only a daily counter — no IP,
// cookie or any other personal data is stored.
func (r *EventRegistrationRepository) RecordView(ctx context.Context, eventID string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO event_page_views (event_id, day, views)
		VALUES ($1, (now() AT TIME ZONE 'America/Fortaleza')::date, 1)
		ON CONFLICT (event_id, day) DO UPDATE SET views = event_page_views.views + 1
	`, eventID)
	return err
}

type RegistrationStats struct {
	Views     int `json:"views"`
	Confirmed int `json:"confirmed"`
	Waitlist  int `json:"waitlist"`
	Cancelled int `json:"cancelled"`
	CheckedIn int `json:"checkedIn"`
	// Capacity is the form limit (nil = sem limite).
	Capacity *int `json:"capacity"`
	// ConversionPct = inscrições (confirmadas + espera) / visitas, em %.
	ConversionPct float64 `json:"conversionPct"`
}

func (r *EventRegistrationRepository) Stats(ctx context.Context, eventID string) (*RegistrationStats, error) {
	var s RegistrationStats
	err := r.pool.QueryRow(ctx, `
		SELECT COALESCE((SELECT SUM(views) FROM event_page_views WHERE event_id = $1), 0),
		       COUNT(*) FILTER (WHERE status = 'confirmed'),
		       COUNT(*) FILTER (WHERE status = 'waitlist'),
		       COUNT(*) FILTER (WHERE status = 'cancelled'),
		       COUNT(*) FILTER (WHERE checked_in_at IS NOT NULL),
		       (SELECT form_capacity FROM events WHERE id = $1)
		FROM event_registrations WHERE event_id = $1
	`, eventID).Scan(&s.Views, &s.Confirmed, &s.Waitlist, &s.Cancelled, &s.CheckedIn, &s.Capacity)
	if err != nil {
		return nil, err
	}
	if s.Views > 0 {
		s.ConversionPct = float64(s.Confirmed+s.Waitlist) * 100 / float64(s.Views)
	}
	return &s, nil
}

type EventRegistration struct {
	ID              string  `json:"id"`
	FullName        string  `json:"fullName"`
	WhatsApp        string  `json:"whatsapp"`
	Email           string  `json:"email"`
	PracticeLevel   string  `json:"practiceLevel"`
	Interest        string  `json:"interest"`
	WeightKg        *int    `json:"weightKg"`
	Status          string  `json:"status"`
	CheckedInAt     *string `json:"checkedInAt"`
	CreatedAt       string  `json:"createdAt"`
	RulesAcceptedAt string  `json:"rulesAcceptedAt"`
	ConsentAt       string  `json:"consentAt"`
}

// List returns one event's sign-ups — confirmed first, then the waitlist in arrival
// order, then cancelled. search matches name, e-mail or WhatsApp.
func (r *EventRegistrationRepository) List(ctx context.Context, eventID, search string) ([]EventRegistration, error) {
	search = strings.TrimSpace(search)
	rows, err := r.pool.Query(ctx, `
		SELECT id, full_name, whatsapp, email, practice_level, interest, weight_kg, status,
		       to_char(checked_in_at, 'YYYY-MM-DD"T"HH24:MI:SSZ'),
		       to_char(created_at, 'YYYY-MM-DD"T"HH24:MI:SSZ'),
		       to_char(rules_accepted_at, 'YYYY-MM-DD"T"HH24:MI:SSZ'),
		       to_char(consent_at, 'YYYY-MM-DD"T"HH24:MI:SSZ')
		FROM event_registrations
		WHERE event_id = $1
		  AND ($2 = '' OR full_name ILIKE $3 OR email ILIKE $3 OR whatsapp ILIKE $3)
		ORDER BY CASE status WHEN 'confirmed' THEN 0 WHEN 'waitlist' THEN 1 ELSE 2 END, created_at
	`, eventID, search, "%"+search+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]EventRegistration, 0)
	for rows.Next() {
		var e EventRegistration
		if err := rows.Scan(&e.ID, &e.FullName, &e.WhatsApp, &e.Email, &e.PracticeLevel, &e.Interest, &e.WeightKg, &e.Status,
			&e.CheckedInAt, &e.CreatedAt, &e.RulesAcceptedAt, &e.ConsentAt); err != nil {
			return nil, err
		}
		list = append(list, e)
	}
	return list, rows.Err()
}

// SetStatus moves a sign-up between confirmed / waitlist / cancelled. Scoped to eventID
// so a stale screen of another event can't touch it.
func (r *EventRegistrationRepository) SetStatus(ctx context.Context, eventID, id, status string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE event_registrations SET status = $1, updated_at = now() WHERE id = $2 AND event_id = $3
	`, status, id, eventID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// SetCheckedIn marks (or unmarks) a confirmed sign-up as present on the event day.
func (r *EventRegistrationRepository) SetCheckedIn(ctx context.Context, eventID, id string, present bool) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE event_registrations
		SET checked_in_at = CASE WHEN $1 THEN COALESCE(checked_in_at, now()) ELSE NULL END, updated_at = now()
		WHERE id = $2 AND event_id = $3
	`, present, id, eventID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
