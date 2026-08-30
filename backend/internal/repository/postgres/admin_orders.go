package postgres

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AdminOrderRepository struct {
	pool *pgxpool.Pool
}

func NewAdminOrderRepository(pool *pgxpool.Pool) *AdminOrderRepository {
	return &AdminOrderRepository{pool: pool}
}

type AdminOrderSummary struct {
	ID           string `json:"id"`
	OrderNumber  string `json:"orderNumber"`
	Status       string `json:"status"`
	TotalCents   int    `json:"totalCents"`
	StudentName  string `json:"studentName"`
	StudentEmail string `json:"studentEmail"`
	CreatedAt    string `json:"createdAt"`
}

// List is the "cliente diz que pagou, cadê o pedido" lookup for staff: filter by status,
// free-text search across order number/student name/e-mail, and/or a date range bounding
// o.created_at (when the order itself was placed).
func (r *AdminOrderRepository) List(ctx context.Context, status, search string, from, to *time.Time) ([]AdminOrderSummary, error) {
	query := `
		SELECT o.id, o.order_number, o.status, o.total_cents, s.full_name, s.email,
		       to_char(o.created_at, 'YYYY-MM-DD"T"HH24:MI:SSZ')
		FROM orders o JOIN students s ON s.id = o.student_id
		WHERE ($1 = '' OR o.status = $1)
		  AND ($2 = '' OR o.order_number ILIKE $3 OR s.full_name ILIKE $3 OR s.email ILIKE $3)
		  AND ($4::timestamptz IS NULL OR o.created_at >= $4)
		  AND ($5::timestamptz IS NULL OR o.created_at < $5::timestamptz + INTERVAL '1 day')
		ORDER BY o.created_at DESC
		LIMIT 100
	`
	rows, err := r.pool.Query(ctx, query, status, search, "%"+strings.TrimSpace(search)+"%", from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]AdminOrderSummary, 0)
	for rows.Next() {
		var o AdminOrderSummary
		if err := rows.Scan(&o.ID, &o.OrderNumber, &o.Status, &o.TotalCents, &o.StudentName, &o.StudentEmail, &o.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, o)
	}
	return list, rows.Err()
}

type AdminOrderItem struct {
	ProductTitle    string  `json:"productTitle"`
	ActivityTitle   *string `json:"activityTitle"`
	BenefitType     string  `json:"benefitType"`
	SessionStartsAt *string `json:"sessionStartsAt"`
}

type AdminOrderEntitlement struct {
	ID                     string  `json:"id"`
	Label                  string  `json:"label"`
	VendorName             string  `json:"vendorName"`
	Status                 string  `json:"status"`
	ValidFrom              string  `json:"validFrom"`
	ValidUntil             string  `json:"validUntil"`
	IssuedAt               string  `json:"issuedAt"`
	UsedAt                 *string `json:"usedAt"`
	UsedByName             string  `json:"usedByName,omitempty"`
	NoShowRescheduleUsedAt *string `json:"noShowRescheduleUsedAt,omitempty"`
}

type AdminOrderReschedule struct {
	Reason        string `json:"reason"`
	PreviousDate  string `json:"previousDate"`
	NewDate       string `json:"newDate"`
	ChangedByName string `json:"changedByName"`
	CreatedAt     string `json:"createdAt"`
}

type AdminOrderDetail struct {
	AdminOrderSummary
	AsaasPaymentID string                  `json:"asaasPaymentId"`
	Items          []AdminOrderItem        `json:"items"`
	Entitlements   []AdminOrderEntitlement `json:"entitlements"`
	Reschedules    []AdminOrderReschedule  `json:"reschedules"`
}

func (r *AdminOrderRepository) Detail(ctx context.Context, orderID string) (*AdminOrderDetail, error) {
	var d AdminOrderDetail
	err := r.pool.QueryRow(ctx, `
		SELECT o.id, o.order_number, o.status, o.total_cents, s.full_name, s.email,
		       to_char(o.created_at, 'YYYY-MM-DD"T"HH24:MI:SSZ'), COALESCE(o.asaas_payment_id, '')
		FROM orders o JOIN students s ON s.id = o.student_id
		WHERE o.id = $1
	`, orderID).Scan(&d.ID, &d.OrderNumber, &d.Status, &d.TotalCents, &d.StudentName, &d.StudentEmail, &d.CreatedAt, &d.AsaasPaymentID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}

	itemRows, err := r.pool.Query(ctx, `
		SELECT p.title, COALESCE(a.title, 'Café da Manhã'), oi.benefit_type, to_char(cs.starts_at, 'YYYY-MM-DD"T"HH24:MI:SSZ')
		FROM order_items oi
		JOIN products p ON p.id = oi.product_id
		LEFT JOIN activities a ON a.id = oi.activity_id
		LEFT JOIN class_sessions cs ON cs.id = oi.class_session_id
		WHERE oi.order_id = $1
	`, orderID)
	if err != nil {
		return nil, err
	}
	d.Items = make([]AdminOrderItem, 0)
	for itemRows.Next() {
		var it AdminOrderItem
		if err := itemRows.Scan(&it.ProductTitle, &it.ActivityTitle, &it.BenefitType, &it.SessionStartsAt); err != nil {
			itemRows.Close()
			return nil, err
		}
		d.Items = append(d.Items, it)
	}
	itemRows.Close()
	if err := itemRows.Err(); err != nil {
		return nil, err
	}

	entRows, err := r.pool.Query(ctx, `
		SELECT e.id, v.name, e.status, e.valid_from::text, e.valid_until::text,
		       to_char(e.issued_at, 'YYYY-MM-DD"T"HH24:MI:SSZ'),
		       to_char(e.used_at, 'YYYY-MM-DD"T"HH24:MI:SSZ'),
		       COALESCE(tm.name, ''),
		       to_char(e.no_show_reschedule_used_at, 'YYYY-MM-DD"T"HH24:MI:SSZ')
		FROM entitlements e
		JOIN vendors v ON v.id = e.vendor_id
		LEFT JOIN team_members tm ON tm.id = e.used_by
		WHERE e.order_id = $1
		ORDER BY e.issued_at
	`, orderID)
	if err != nil {
		return nil, err
	}
	d.Entitlements = make([]AdminOrderEntitlement, 0)
	ids := make([]string, 0)
	index := map[string]int{}
	for entRows.Next() {
		var e AdminOrderEntitlement
		if err := entRows.Scan(&e.ID, &e.VendorName, &e.Status, &e.ValidFrom, &e.ValidUntil, &e.IssuedAt, &e.UsedAt, &e.UsedByName, &e.NoShowRescheduleUsedAt); err != nil {
			entRows.Close()
			return nil, err
		}
		index[e.ID] = len(d.Entitlements)
		ids = append(ids, e.ID)
		d.Entitlements = append(d.Entitlements, e)
	}
	entRows.Close()
	if err := entRows.Err(); err != nil {
		return nil, err
	}

	if len(ids) > 0 {
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
		labels := map[string][]string{}
		for labelRows.Next() {
			var entitlementID, label string
			if err := labelRows.Scan(&entitlementID, &label); err != nil {
				labelRows.Close()
				return nil, err
			}
			labels[entitlementID] = append(labels[entitlementID], label)
		}
		labelRows.Close()
		if err := labelRows.Err(); err != nil {
			return nil, err
		}
		for id, parts := range labels {
			d.Entitlements[index[id]].Label = strings.Join(parts, " + ")
		}
	}

	reschedRows, err := r.pool.Query(ctx, `
		SELECT orr.reason, orr.previous_date::text, orr.new_date::text,
		       COALESCE(tm.name, s.full_name || ' (cliente)'), to_char(orr.created_at, 'YYYY-MM-DD"T"HH24:MI:SSZ')
		FROM order_reschedules orr
		LEFT JOIN team_members tm ON tm.id = orr.changed_by
		LEFT JOIN students s ON s.id = orr.changed_by_student_id
		WHERE orr.order_id = $1
		ORDER BY orr.created_at DESC
	`, orderID)
	if err != nil {
		return nil, err
	}
	defer reschedRows.Close()
	d.Reschedules = make([]AdminOrderReschedule, 0)
	for reschedRows.Next() {
		var rr AdminOrderReschedule
		if err := reschedRows.Scan(&rr.Reason, &rr.PreviousDate, &rr.NewDate, &rr.ChangedByName, &rr.CreatedAt); err != nil {
			return nil, err
		}
		d.Reschedules = append(d.Reschedules, rr)
	}
	if err := reschedRows.Err(); err != nil {
		return nil, err
	}

	return &d, nil
}
