package postgres

import (
	"context"
	"sort"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type SessionRepository struct {
	pool *pgxpool.Pool
}

func NewSessionRepository(pool *pgxpool.Pool) *SessionRepository {
	return &SessionRepository{pool: pool}
}

type AdminSession struct {
	ID            string    `json:"id"`
	ActivityID    string    `json:"activityId"`
	ActivityTitle string    `json:"activityTitle"`
	StartsAt      time.Time `json:"startsAt"`
	EndsAt        time.Time `json:"endsAt"`
	Capacity      int       `json:"capacity"`
	Booked        int       `json:"booked"`
	Status        string    `json:"status"`
}

// ListUpcoming returns sessions from today onward, across all activities, with how many
// seats are already booked (order_items on paid orders referencing that session).
func (r *SessionRepository) ListUpcoming(ctx context.Context) ([]AdminSession, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT cs.id, cs.activity_id, a.title, cs.starts_at, cs.ends_at, cs.capacity, cs.status,
		       COALESCE((
		         SELECT COUNT(*) FROM order_items oi
		         JOIN orders o ON o.id = oi.order_id
		         WHERE oi.class_session_id = cs.id AND o.status = 'paid'
		       ), 0) AS booked
		FROM class_sessions cs
		JOIN activities a ON a.id = cs.activity_id
		WHERE cs.starts_at >= now() - interval '1 day'
		ORDER BY cs.starts_at ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]AdminSession, 0)
	for rows.Next() {
		var s AdminSession
		if err := rows.Scan(&s.ID, &s.ActivityID, &s.ActivityTitle, &s.StartsAt, &s.EndsAt, &s.Capacity, &s.Status, &s.Booked); err != nil {
			return nil, err
		}
		list = append(list, s)
	}
	return list, rows.Err()
}

func (r *SessionRepository) Create(ctx context.Context, activityID string, startsAt, endsAt time.Time, capacity int) (string, error) {
	var id string
	err := r.pool.QueryRow(ctx, `
		INSERT INTO class_sessions (activity_id, starts_at, ends_at, capacity)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, activityID, startsAt, endsAt, capacity).Scan(&id)
	return id, err
}

func (r *SessionRepository) SetStatus(ctx context.Context, id, status string) error {
	_, err := r.pool.Exec(ctx, `UPDATE class_sessions SET status = $1, updated_at = now() WHERE id = $2`, status, id)
	return err
}

// Booked returns how many paid+pending order_items already reference this session —
// used to stop an admin from setting a new capacity below what's already booked.
func (r *SessionRepository) Booked(ctx context.Context, id string) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM order_items oi
		JOIN orders o ON o.id = oi.order_id
		WHERE oi.class_session_id = $1 AND o.status IN ('paid', 'pending')
	`, id).Scan(&n)
	return n, err
}

func (r *SessionRepository) Update(ctx context.Context, id string, startsAt, endsAt time.Time, capacity int) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE class_sessions SET starts_at = $1, ends_at = $2, capacity = $3, updated_at = now() WHERE id = $4
	`, startsAt, endsAt, capacity, id)
	return err
}

// AvailableForActivity returns upcoming, non-cancelled sessions for one activity with
// open seats — what the public checkout offers as pickable dates.
func (r *SessionRepository) AvailableForActivity(ctx context.Context, activityID string) ([]AdminSession, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT cs.id, cs.activity_id, a.title, cs.starts_at, cs.ends_at, cs.capacity, cs.status,
		       COALESCE(booked.n, 0) AS booked
		FROM class_sessions cs
		JOIN activities a ON a.id = cs.activity_id
		LEFT JOIN (
		  SELECT oi.class_session_id, COUNT(*) AS n
		  FROM order_items oi
		  JOIN orders o ON o.id = oi.order_id
		  WHERE o.status IN ('paid', 'pending')
		  GROUP BY oi.class_session_id
		) booked ON booked.class_session_id = cs.id
		WHERE cs.activity_id = $1
		  AND cs.status = 'scheduled'
		  AND cs.starts_at >= now()
		  AND COALESCE(booked.n, 0) < cs.capacity
		ORDER BY cs.starts_at ASC
	`, activityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]AdminSession, 0)
	for rows.Next() {
		var s AdminSession
		if err := rows.Scan(&s.ID, &s.ActivityID, &s.ActivityTitle, &s.StartsAt, &s.EndsAt, &s.Capacity, &s.Status, &s.Booked); err != nil {
			return nil, err
		}
		list = append(list, s)
	}
	return list, rows.Err()
}

// NextPerActivity returns the soonest upcoming, open-seat session for each active
// activity — what the public landing page shows so a visitor knows the next event date
// before ever going to checkout, instead of only discovering it after picking a product.
func (r *SessionRepository) NextPerActivity(ctx context.Context) ([]AdminSession, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT DISTINCT ON (cs.activity_id) cs.id, cs.activity_id, a.title, cs.starts_at, cs.ends_at, cs.capacity, cs.status,
		       COALESCE(booked.n, 0) AS booked
		FROM class_sessions cs
		JOIN activities a ON a.id = cs.activity_id
		LEFT JOIN (
		  SELECT oi.class_session_id, COUNT(*) AS n
		  FROM order_items oi
		  JOIN orders o ON o.id = oi.order_id
		  WHERE o.status IN ('paid', 'pending')
		  GROUP BY oi.class_session_id
		) booked ON booked.class_session_id = cs.id
		WHERE a.active = true
		  AND cs.status = 'scheduled'
		  AND cs.starts_at >= now()
		  AND COALESCE(booked.n, 0) < cs.capacity
		ORDER BY cs.activity_id, cs.starts_at ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]AdminSession, 0)
	for rows.Next() {
		var s AdminSession
		if err := rows.Scan(&s.ID, &s.ActivityID, &s.ActivityTitle, &s.StartsAt, &s.EndsAt, &s.Capacity, &s.Status, &s.Booked); err != nil {
			return nil, err
		}
		list = append(list, s)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].StartsAt.Before(list[j].StartsAt) })
	return list, rows.Err()
}

// HasCapacity checks a single session still has an open seat — used as the final,
// authoritative check inside the checkout transaction (the listing above can go stale
// between the customer loading the page and confirming).
func (r *SessionRepository) HasCapacity(ctx context.Context, sessionID string) (bool, error) {
	var capacity, booked int
	err := r.pool.QueryRow(ctx, `
		SELECT cs.capacity, COALESCE((
			SELECT COUNT(*) FROM order_items oi
			JOIN orders o ON o.id = oi.order_id
			WHERE oi.class_session_id = cs.id AND o.status IN ('paid', 'pending')
		), 0)
		FROM class_sessions cs WHERE cs.id = $1 AND cs.status = 'scheduled'
	`, sessionID).Scan(&capacity, &booked)
	if err != nil {
		return false, err
	}
	return booked < capacity, nil
}
