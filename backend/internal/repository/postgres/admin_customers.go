package postgres

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AdminCustomerRepository struct {
	pool *pgxpool.Pool
}

func NewAdminCustomerRepository(pool *pgxpool.Pool) *AdminCustomerRepository {
	return &AdminCustomerRepository{pool: pool}
}

type AdminCustomerSummary struct {
	ID          string `json:"id"`
	FullName    string `json:"fullName"`
	Email       string `json:"email"`
	Phone       string `json:"phone"`
	CPFLast4    string `json:"cpfLast4"`
	OrdersCount int    `json:"ordersCount"`
	CreatedAt   string `json:"createdAt"`
}

// Search is the "cliente diz que pagou mas não achou o e-mail" lookup: name, e-mail,
// phone (partial match) or the last 4 digits of the CPF collected at registration —
// exactly the redundancy data that was collected for this purpose. An empty query lists
// every customer instead of filtering, so the admin panel can show the full base by
// default rather than starting on a "type something to search" dead end.
func (r *AdminCustomerRepository) Search(ctx context.Context, query string) ([]AdminCustomerSummary, error) {
	limit := 50
	pattern := "%" + query + "%"
	if query == "" {
		limit = 200
	}
	rows, err := r.pool.Query(ctx, `
		SELECT s.id, s.full_name, s.email, COALESCE(s.phone, ''), COALESCE(s.cpf_last4, ''),
		       COALESCE((SELECT COUNT(*) FROM orders o WHERE o.student_id = s.id), 0),
		       to_char(s.created_at, 'YYYY-MM-DD"T"HH24:MI:SSZ')
		FROM students s
		WHERE s.deleted_at IS NULL
		  AND ($1 = '' OR s.full_name ILIKE $2 OR s.email ILIKE $2 OR s.phone ILIKE $2 OR s.cpf_last4 = $1)
		ORDER BY s.created_at DESC
		LIMIT $3
	`, query, pattern, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]AdminCustomerSummary, 0)
	for rows.Next() {
		var c AdminCustomerSummary
		if err := rows.Scan(&c.ID, &c.FullName, &c.Email, &c.Phone, &c.CPFLast4, &c.OrdersCount, &c.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, rows.Err()
}

type AdminCustomerDetail struct {
	AdminCustomerSummary
	Orders []AdminCustomerOrder `json:"orders"`
}

// AdminCustomerOrder carries everything a staff member needs to resolve a "I paid
// but..." support conversation without leaving the customer detail modal: payment
// timing/method alongside every QR ticket (AdminOrderEntitlement, shared with the
// Pedidos order-detail view) issued from the order and its exact validation state.
type AdminCustomerOrder struct {
	ID            string                  `json:"id"`
	OrderNumber   string                  `json:"orderNumber"`
	Status        string                  `json:"status"`
	TotalCents    int                     `json:"totalCents"`
	PaymentMethod string                  `json:"paymentMethod"`
	ProductTitle  string                  `json:"productTitle"`
	CreatedAt     string                  `json:"createdAt"`
	PaidAt        string                  `json:"paidAt,omitempty"`
	Tickets       []AdminOrderEntitlement `json:"tickets"`
}

func (r *AdminCustomerRepository) Detail(ctx context.Context, id string) (*AdminCustomerDetail, error) {
	var c AdminCustomerSummary
	err := r.pool.QueryRow(ctx, `
		SELECT s.id, s.full_name, s.email, COALESCE(s.phone, ''), COALESCE(s.cpf_last4, ''),
		       COALESCE((SELECT COUNT(*) FROM orders o WHERE o.student_id = s.id), 0),
		       to_char(s.created_at, 'YYYY-MM-DD"T"HH24:MI:SSZ')
		FROM students s WHERE s.id = $1 AND s.deleted_at IS NULL
	`, id).Scan(&c.ID, &c.FullName, &c.Email, &c.Phone, &c.CPFLast4, &c.OrdersCount, &c.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}

	orders, err := r.ordersWithTickets(ctx, id)
	if err != nil {
		return nil, err
	}

	return &AdminCustomerDetail{AdminCustomerSummary: c, Orders: orders}, nil
}

func (r *AdminCustomerRepository) ordersWithTickets(ctx context.Context, studentID string) ([]AdminCustomerOrder, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT o.id, o.order_number, o.status, o.total_cents, COALESCE(o.payment_method, ''),
		       COALESCE((SELECT p.title FROM order_items oi JOIN products p ON p.id = oi.product_id WHERE oi.order_id = o.id LIMIT 1), ''),
		       to_char(o.created_at, 'YYYY-MM-DD"T"HH24:MI:SSZ'),
		       COALESCE(to_char(o.paid_at, 'YYYY-MM-DD"T"HH24:MI:SSZ'), '')
		FROM orders o
		WHERE o.student_id = $1
		ORDER BY o.created_at DESC
	`, studentID)
	if err != nil {
		return nil, err
	}
	orders := make([]AdminCustomerOrder, 0)
	index := map[string]int{}
	for rows.Next() {
		var o AdminCustomerOrder
		if err := rows.Scan(&o.ID, &o.OrderNumber, &o.Status, &o.TotalCents, &o.PaymentMethod, &o.ProductTitle, &o.CreatedAt, &o.PaidAt); err != nil {
			rows.Close()
			return nil, err
		}
		o.Tickets = []AdminOrderEntitlement{}
		index[o.ID] = len(orders)
		orders = append(orders, o)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()
	if len(orders) == 0 {
		return orders, nil
	}

	ticketRows, err := r.pool.Query(ctx, `
		SELECT e.id, e.order_id, e.status, v.name, e.valid_from::text, e.valid_until::text,
		       to_char(e.issued_at, 'YYYY-MM-DD"T"HH24:MI:SSZ'),
		       to_char(e.used_at, 'YYYY-MM-DD"T"HH24:MI:SSZ'),
		       COALESCE(tm.name, '')
		FROM entitlements e
		JOIN vendors v ON v.id = e.vendor_id
		LEFT JOIN team_members tm ON tm.id = e.used_by
		WHERE e.student_id = $1
		ORDER BY e.issued_at DESC
	`, studentID)
	if err != nil {
		return nil, err
	}
	defer ticketRows.Close()

	type ticketRow struct {
		orderID string
		t       AdminOrderEntitlement
	}
	var scanned []ticketRow
	ticketIDs := make([]string, 0)
	for ticketRows.Next() {
		var tr ticketRow
		if err := ticketRows.Scan(&tr.t.ID, &tr.orderID, &tr.t.Status, &tr.t.VendorName, &tr.t.ValidFrom, &tr.t.ValidUntil, &tr.t.IssuedAt, &tr.t.UsedAt, &tr.t.UsedByName); err != nil {
			return nil, err
		}
		scanned = append(scanned, tr)
		ticketIDs = append(ticketIDs, tr.t.ID)
	}
	if err := ticketRows.Err(); err != nil {
		return nil, err
	}
	if len(ticketIDs) == 0 {
		return orders, nil
	}

	labelRows, err := r.pool.Query(ctx, `
		SELECT ei.entitlement_id, COALESCE(a.title, 'Café da Manhã')
		FROM entitlement_items ei
		JOIN order_items oi ON oi.id = ei.order_item_id
		LEFT JOIN activities a ON a.id = oi.activity_id
		WHERE ei.entitlement_id = ANY($1)
	`, ticketIDs)
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

	for _, tr := range scanned {
		tr.t.Label = strings.Join(labels[tr.t.ID], " + ")
		if oi, ok := index[tr.orderID]; ok {
			orders[oi].Tickets = append(orders[oi].Tickets, tr.t)
		}
	}

	return orders, nil
}
