package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DefaultEventID is the event that was already on sale before events existed (seeded by
// 00029_events.sql). It keeps the "/" page and "/comprar", and every row inserted
// without an explicit event_id falls into it through the column DEFAULT.
const DefaultEventID = "00000000-0000-0000-0000-0000000000e1"

var ErrEventSlugTaken = errors.New("já existe um evento com esse endereço")
var ErrDefaultEventLocked = errors.New("o evento principal não pode ser desativado")
var ErrOtherEventActivity = errors.New("atividade pertence a outro evento")

type Event struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Slug          string `json:"slug"`
	IsDefault     bool   `json:"isDefault"`
	Active        bool   `json:"active"`
	Headline      string `json:"headline"`
	Subtitle      string `json:"subtitle"`
	DateLabel     string `json:"dateLabel"`
	LocationLabel string `json:"locationLabel"`
	Description   string `json:"description"`
	BannerURL     string `json:"bannerUrl"`
	Includes      string `json:"includes"`
	Schedule      string `json:"schedule"`
	DisplayOrder  int    `json:"displayOrder"`
}

// EventInput is what the Admin edits. IsDefault and Active are not part of it: the
// default event is fixed by the migration, and publishing goes through SetActive.
type EventInput struct {
	Name          string
	Slug          string
	Headline      string
	Subtitle      string
	DateLabel     string
	LocationLabel string
	Description   string
	BannerURL     string
	Includes      string
	Schedule      string
}

type EventRepository struct {
	pool *pgxpool.Pool
}

func NewEventRepository(pool *pgxpool.Pool) *EventRepository {
	return &EventRepository{pool: pool}
}

const eventColumns = `id, name, slug, is_default, active,
	COALESCE(headline, ''), COALESCE(subtitle, ''), COALESCE(date_label, ''), COALESCE(location_label, ''),
	COALESCE(description, ''), COALESCE(banner_url, ''), COALESCE(includes, ''), COALESCE(schedule, ''), display_order`

func scanEvent(row pgx.Row) (*Event, error) {
	var e Event
	err := row.Scan(&e.ID, &e.Name, &e.Slug, &e.IsDefault, &e.Active,
		&e.Headline, &e.Subtitle, &e.DateLabel, &e.LocationLabel,
		&e.Description, &e.BannerURL, &e.Includes, &e.Schedule, &e.DisplayOrder)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &e, nil
}

// List returns every event, the default one first — the order the Admin selector shows.
func (r *EventRepository) List(ctx context.Context) ([]Event, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+eventColumns+` FROM events ORDER BY is_default DESC, display_order, created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]Event, 0)
	for rows.Next() {
		e, err := scanEvent(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, *e)
	}
	return list, rows.Err()
}

func (r *EventRepository) GetByID(ctx context.Context, id string) (*Event, error) {
	return scanEvent(r.pool.QueryRow(ctx, `SELECT `+eventColumns+` FROM events WHERE id = $1`, id))
}

// GetActiveBySlug is the public lookup behind /<slug>: an unpublished event is reported
// as not found, so its page and its products stay invisible until the Admin publishes it.
func (r *EventRepository) GetActiveBySlug(ctx context.Context, slug string) (*Event, error) {
	return scanEvent(r.pool.QueryRow(ctx, `SELECT `+eventColumns+` FROM events WHERE slug = $1 AND active = true`, slug))
}

// ResolvePublic maps the ?event=<slug> of a public request to an event id. No slug means
// the default event, which is what "/" and "/comprar" have always shown.
func (r *EventRepository) ResolvePublic(ctx context.Context, slug string) (string, error) {
	if slug == "" {
		return DefaultEventID, nil
	}
	e, err := r.GetActiveBySlug(ctx, slug)
	if err != nil {
		return "", err
	}
	return e.ID, nil
}

// NameTaken reports whether another event (other than exceptID) already uses this name —
// two events with the same name in the selector would invite exactly the mix-ups the
// event separation exists to prevent.
func (r *EventRepository) NameTaken(ctx context.Context, name, exceptID string) (bool, error) {
	var taken bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM events
			WHERE lower(btrim(name)) = lower(btrim($1)) AND ($2::uuid IS NULL OR id <> $2::uuid)
		)
	`, name, nullableEventID(exceptID)).Scan(&taken)
	return taken, err
}

func (r *EventRepository) Exists(ctx context.Context, id string) (bool, error) {
	var ok bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM events WHERE id = $1)`, id).Scan(&ok)
	return ok, err
}

// Create adds a new event, unpublished: the Admin fills in products, turmas and the page
// before switching it on, so a half-configured event never shows up on the site.
func (r *EventRepository) Create(ctx context.Context, in EventInput) (string, error) {
	var id string
	err := r.pool.QueryRow(ctx, `
		INSERT INTO events (name, slug, headline, subtitle, date_label, location_label, description, banner_url, includes, schedule, display_order)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, (SELECT COALESCE(MAX(display_order), 0) + 1 FROM events))
		RETURNING id
	`, in.Name, in.Slug, nullIfEmpty(in.Headline), nullIfEmpty(in.Subtitle), nullIfEmpty(in.DateLabel), nullIfEmpty(in.LocationLabel),
		nullIfEmpty(in.Description), nullIfEmpty(in.BannerURL), nullIfEmpty(in.Includes), nullIfEmpty(in.Schedule)).Scan(&id)
	if isUniqueViolation(err) {
		return "", ErrEventSlugTaken
	}
	return id, err
}

func (r *EventRepository) Update(ctx context.Context, id string, in EventInput) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE events
		SET name = $1, slug = $2, headline = $3, subtitle = $4, date_label = $5, location_label = $6,
		    description = $7, banner_url = $8, includes = $9, schedule = $10, updated_at = now()
		WHERE id = $11
	`, in.Name, in.Slug, nullIfEmpty(in.Headline), nullIfEmpty(in.Subtitle), nullIfEmpty(in.DateLabel), nullIfEmpty(in.LocationLabel),
		nullIfEmpty(in.Description), nullIfEmpty(in.BannerURL), nullIfEmpty(in.Includes), nullIfEmpty(in.Schedule), id)
	if isUniqueViolation(err) {
		return ErrEventSlugTaken
	}
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *EventRepository) SetActive(ctx context.Context, id string, active bool) error {
	if id == DefaultEventID && !active {
		return ErrDefaultEventLocked
	}
	tag, err := r.pool.Exec(ctx, `UPDATE events SET active = $1, updated_at = now() WHERE id = $2 AND (is_default = false OR $1 = true)`, active, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// EventIDForActivity is used to refuse cross-event writes, e.g. a turma created in the
// Admin of event B for an activity that belongs to event A.
func (r *EventRepository) EventIDForActivity(ctx context.Context, activityID string) (string, error) {
	var id string
	err := r.pool.QueryRow(ctx, `SELECT event_id FROM activities WHERE id = $1`, activityID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return id, err
}

// EventIDForProduct is EventIDForActivity for a product.
func (r *EventRepository) EventIDForProduct(ctx context.Context, productID string) (string, error) {
	return r.lookupEventID(ctx, `SELECT event_id FROM products WHERE id = $1`, productID)
}

// EventIDForVoucher is EventIDForActivity for a voucher.
func (r *EventRepository) EventIDForVoucher(ctx context.Context, voucherID string) (string, error) {
	return r.lookupEventID(ctx, `SELECT event_id FROM vouchers WHERE id = $1`, voucherID)
}

// EventIDForOrder is EventIDForActivity for an order.
func (r *EventRepository) EventIDForOrder(ctx context.Context, orderID string) (string, error) {
	return r.lookupEventID(ctx, `SELECT event_id FROM orders WHERE id = $1`, orderID)
}

// EventIDForEntitlement is EventIDForActivity for a ticket (through its order).
func (r *EventRepository) EventIDForEntitlement(ctx context.Context, entitlementID string) (string, error) {
	return r.lookupEventID(ctx, `SELECT o.event_id FROM entitlements e JOIN orders o ON o.id = e.order_id WHERE e.id = $1`, entitlementID)
}

func (r *EventRepository) lookupEventID(ctx context.Context, query, id string) (string, error) {
	var eventID string
	err := r.pool.QueryRow(ctx, query, id).Scan(&eventID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return eventID, err
}

// EventIDForSession is EventIDForActivity for an existing turma.
func (r *EventRepository) EventIDForSession(ctx context.Context, sessionID string) (string, error) {
	var id string
	err := r.pool.QueryRow(ctx, `
		SELECT a.event_id FROM class_sessions cs JOIN activities a ON a.id = cs.activity_id WHERE cs.id = $1
	`, sessionID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return id, err
}

// nullableEventID turns the "" (no event filter) used across the handlers into the NULL
// that the `($n::uuid IS NULL OR x.event_id = $n::uuid)` filters expect.
func nullableEventID(eventID string) *string {
	if eventID == "" {
		return nil
	}
	return &eventID
}
