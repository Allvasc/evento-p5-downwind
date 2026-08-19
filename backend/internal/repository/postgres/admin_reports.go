package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AdminReportsRepository struct {
	pool *pgxpool.Pool
}

func NewAdminReportsRepository(pool *pgxpool.Pool) *AdminReportsRepository {
	return &AdminReportsRepository{pool: pool}
}

type Attendee struct {
	Name        string `json:"name"`
	Email       string `json:"email"`
	OrderNumber string `json:"orderNumber"`
}

type SessionRoster struct {
	SessionID     string     `json:"sessionId"`
	ActivityTitle string     `json:"activityTitle"`
	StartsAt      string     `json:"startsAt"`
	Capacity      int        `json:"capacity"`
	Attendees     []Attendee `json:"attendees"`
}

// SessionsWithAttendees lists every upcoming scheduled turma with the names of
// everyone who paid for a seat in it — the roster staff needs to know who to expect at
// each specific date/time, independent of which product they bought it through. from/to
// bound cs.starts_at (the turma's own date) — either may be nil to leave that side open.
func (r *AdminReportsRepository) SessionsWithAttendees(ctx context.Context, from, to *time.Time) ([]SessionRoster, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT cs.id, a.title, to_char(cs.starts_at, 'YYYY-MM-DD"T"HH24:MI:SSZ'), cs.capacity,
		       COALESCE(s.full_name, ''), COALESCE(s.email, ''), COALESCE(o.order_number, '')
		FROM class_sessions cs
		JOIN activities a ON a.id = cs.activity_id
		LEFT JOIN order_items oi ON oi.class_session_id = cs.id
		LEFT JOIN orders o ON o.id = oi.order_id AND o.status = 'paid'
		LEFT JOIN students s ON s.id = o.student_id
		WHERE cs.status = 'scheduled'
		  AND ($1::timestamptz IS NULL OR cs.starts_at >= $1)
		  AND ($2::timestamptz IS NULL OR cs.starts_at < $2::timestamptz + INTERVAL '1 day')
		ORDER BY cs.starts_at, a.title, s.full_name
	`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]SessionRoster, 0)
	index := map[string]int{}
	for rows.Next() {
		var sessionID, activityTitle, startsAt, name, email, orderNumber string
		var capacity int
		if err := rows.Scan(&sessionID, &activityTitle, &startsAt, &capacity, &name, &email, &orderNumber); err != nil {
			return nil, err
		}
		i, ok := index[sessionID]
		if !ok {
			i = len(list)
			index[sessionID] = i
			list = append(list, SessionRoster{
				SessionID: sessionID, ActivityTitle: activityTitle, StartsAt: startsAt,
				Capacity: capacity, Attendees: []Attendee{},
			})
		}
		if orderNumber != "" {
			list[i].Attendees = append(list[i].Attendees, Attendee{Name: name, Email: email, OrderNumber: orderNumber})
		}
	}
	return list, rows.Err()
}

type ProductRoster struct {
	ProductID string     `json:"productId"`
	Title     string     `json:"title"`
	Buyers    []Attendee `json:"buyers"`
}

// ProductsWithBuyers lists every active product with the names of everyone who paid
// for it, independent of session/date — e.g. every "Aula Individual — Yoga" buyer in
// one place, across every turma. Products with no buyers yet still appear (empty list),
// and that stays true even when from/to narrow the buyer list to zero for a product —
// the date bound lives in the orders JOIN condition, not a WHERE clause, so it only
// prunes which purchases count as a "buyer" rather than hiding products outright.
func (r *AdminReportsRepository) ProductsWithBuyers(ctx context.Context, from, to *time.Time) ([]ProductRoster, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT DISTINCT p.display_order, p.id, p.title, COALESCE(o.id::text, ''), COALESCE(s.full_name, ''), COALESCE(s.email, ''), COALESCE(o.order_number, ''), o.created_at
		FROM products p
		LEFT JOIN order_items oi ON oi.product_id = p.id
		LEFT JOIN orders o ON o.id = oi.order_id AND o.status = 'paid'
		  AND ($1::timestamptz IS NULL OR o.created_at >= $1)
		  AND ($2::timestamptz IS NULL OR o.created_at < $2::timestamptz + INTERVAL '1 day')
		LEFT JOIN students s ON s.id = o.student_id
		WHERE p.active = true
		ORDER BY p.display_order, o.created_at
	`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type row struct {
		productID, title, orderID, name, email, orderNumber string
	}
	var scanned []row
	for rows.Next() {
		var r row
		var displayOrder int
		var createdAt any
		if err := rows.Scan(&displayOrder, &r.productID, &r.title, &r.orderID, &r.name, &r.email, &r.orderNumber, &createdAt); err != nil {
			return nil, err
		}
		scanned = append(scanned, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Preserve product display order (from the products table's own ORDER BY, applied
	// via a second pass keyed on first-seen order) while deduping nothing further —
	// SELECT DISTINCT already collapsed the one-row-per-order_item duplication.
	list := make([]ProductRoster, 0)
	index := map[string]int{}
	for _, r := range scanned {
		i, ok := index[r.productID]
		if !ok {
			i = len(list)
			index[r.productID] = i
			list = append(list, ProductRoster{ProductID: r.productID, Title: r.title, Buyers: []Attendee{}})
		}
		if r.orderNumber != "" {
			list[i].Buyers = append(list[i].Buyers, Attendee{Name: r.name, Email: r.email, OrderNumber: r.orderNumber})
		}
	}
	return list, nil
}

type ActivityRoster struct {
	Label     string     `json:"label"`
	Attendees []Attendee `json:"attendees"`
}

// ByActivity collapses attendance to the underlying benefit itself — every Yoga booking
// in one place, every HYROX booking in another, every Café da Manhã in a third —
// independent of which product or specific turma/date it was bought through. A single
// combo purchase (e.g. "Aulas" = Yoga + HYROX) contributes one order_item per activity,
// so the same buyer can appear once under Yoga and once under HYROX. from/to bound
// o.created_at (the purchase date, since activities/order_items have no date of their own).
func (r *AdminReportsRepository) ByActivity(ctx context.Context, from, to *time.Time) ([]ActivityRoster, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT COALESCE(a.title, 'Café da Manhã') AS label,
		       COALESCE(s.full_name, ''), COALESCE(s.email, ''), COALESCE(o.order_number, '')
		FROM order_items oi
		JOIN orders o ON o.id = oi.order_id AND o.status = 'paid'
		JOIN students s ON s.id = o.student_id
		LEFT JOIN activities a ON a.id = oi.activity_id
		WHERE ($1::timestamptz IS NULL OR o.created_at >= $1)
		  AND ($2::timestamptz IS NULL OR o.created_at < $2::timestamptz + INTERVAL '1 day')
		ORDER BY label, s.full_name
	`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]ActivityRoster, 0)
	index := map[string]int{}
	for rows.Next() {
		var label, name, email, orderNumber string
		if err := rows.Scan(&label, &name, &email, &orderNumber); err != nil {
			return nil, err
		}
		i, ok := index[label]
		if !ok {
			i = len(list)
			index[label] = i
			list = append(list, ActivityRoster{Label: label, Attendees: []Attendee{}})
		}
		list[i].Attendees = append(list[i].Attendees, Attendee{Name: name, Email: email, OrderNumber: orderNumber})
	}
	return list, rows.Err()
}
