package postgres

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type StudentAreaRepository struct {
	pool *pgxpool.Pool
}

func NewStudentAreaRepository(pool *pgxpool.Pool) *StudentAreaRepository {
	return &StudentAreaRepository{pool: pool}
}

type OrderSummary struct {
	ID           string
	OrderNumber  string
	Status       string
	TotalCents   int
	ProductTitle string
	CreatedAt    string
}

func (r *StudentAreaRepository) ListOrders(ctx context.Context, studentID string) ([]OrderSummary, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT o.id, o.order_number, o.status, o.total_cents,
		       COALESCE((SELECT p.title FROM order_items oi JOIN products p ON p.id = oi.product_id WHERE oi.order_id = o.id LIMIT 1), ''),
		       to_char(o.created_at, 'YYYY-MM-DD"T"HH24:MI:SSZ')
		FROM orders o
		WHERE o.student_id = $1
		ORDER BY o.created_at DESC
	`, studentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	summaries := make([]OrderSummary, 0)
	for rows.Next() {
		var s OrderSummary
		if err := rows.Scan(&s.ID, &s.OrderNumber, &s.Status, &s.TotalCents, &s.ProductTitle, &s.CreatedAt); err != nil {
			return nil, err
		}
		summaries = append(summaries, s)
	}
	return summaries, rows.Err()
}

type TicketSummary struct {
	ID          string
	Status      string
	Label       string
	VendorName  string
	OrderNumber string
	ValidFrom   string
	ValidUntil  string
	UsedAt      string
}

func (r *StudentAreaRepository) ListTickets(ctx context.Context, studentID string) ([]TicketSummary, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT e.id, e.status, v.name, o.order_number, e.valid_from::text, e.valid_until::text,
		       COALESCE(to_char(e.used_at, 'YYYY-MM-DD"T"HH24:MI:SSZ'), '')
		FROM entitlements e
		JOIN orders o ON o.id = e.order_id
		JOIN vendors v ON v.id = e.vendor_id
		WHERE e.student_id = $1
		ORDER BY e.issued_at DESC
	`, studentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tickets := make([]TicketSummary, 0)
	ids := make([]string, 0)
	index := map[string]int{}
	for rows.Next() {
		var t TicketSummary
		if err := rows.Scan(&t.ID, &t.Status, &t.VendorName, &t.OrderNumber, &t.ValidFrom, &t.ValidUntil, &t.UsedAt); err != nil {
			return nil, err
		}
		index[t.ID] = len(tickets)
		ids = append(ids, t.ID)
		tickets = append(tickets, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return tickets, nil
	}

	labelRows, err := r.pool.Query(ctx, `
		SELECT ei.entitlement_id, COALESCE(a.title, 'Café da Manhã')
		FROM entitlement_items ei
		JOIN order_items oi ON oi.id = ei.order_item_id
		LEFT JOIN activities a ON a.id = oi.activity_id
		WHERE ei.entitlement_id = ANY($1)
	`, ids)
	if err != nil {
		return nil, err
	}
	defer labelRows.Close()

	labels := map[string][]string{}
	for labelRows.Next() {
		var entitlementID, label string
		if err := labelRows.Scan(&entitlementID, &label); err != nil {
			return nil, err
		}
		labels[entitlementID] = append(labels[entitlementID], label)
	}
	if err := labelRows.Err(); err != nil {
		return nil, err
	}
	for id, parts := range labels {
		tickets[index[id]].Label = strings.Join(parts, " + ")
	}

	return tickets, nil
}

// TokenForOwnedTicket returns the QR token for an entitlement, but only if it belongs
// to the given student — used to render a ticket's QR image without leaking others'.
func (r *StudentAreaRepository) TokenForOwnedTicket(ctx context.Context, studentID, entitlementID string) (string, error) {
	var token string
	err := r.pool.QueryRow(ctx, `
		SELECT qr_token FROM entitlements WHERE id = $1 AND student_id = $2
	`, entitlementID, studentID).Scan(&token)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return token, err
}
