package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DashboardRepository struct {
	pool *pgxpool.Pool
}

func NewDashboardRepository(pool *pgxpool.Pool) *DashboardRepository {
	return &DashboardRepository{pool: pool}
}

type DashboardSummary struct {
	TotalRevenueCents  int `json:"totalRevenueCents"`
	PaidOrders         int `json:"paidOrders"`
	EntitlementsIssued int `json:"entitlementsIssued"`
	EntitlementsUsed   int `json:"entitlementsUsed"`
	ActiveStudents     int `json:"activeStudents"`
}

// Summary defaults to all-time totals when from/to are nil. When given, each metric is
// scoped to the date that makes sense for it — orders by paid_at, entitlements by
// issued_at (or used_at for the "used" count) — rather than a single shared date column.
func (r *DashboardRepository) Summary(ctx context.Context, from, to *time.Time) (*DashboardSummary, error) {
	var s DashboardSummary
	err := r.pool.QueryRow(ctx, `
		SELECT
			COALESCE((SELECT SUM(total_cents) FROM orders
				WHERE status = 'paid'
				  AND ($1::timestamptz IS NULL OR paid_at >= $1)
				  AND ($2::timestamptz IS NULL OR paid_at < $2::timestamptz + INTERVAL '1 day')), 0),
			(SELECT COUNT(*) FROM orders
				WHERE status = 'paid'
				  AND ($1::timestamptz IS NULL OR paid_at >= $1)
				  AND ($2::timestamptz IS NULL OR paid_at < $2::timestamptz + INTERVAL '1 day')),
			(SELECT COUNT(*) FROM entitlements
				WHERE ($1::timestamptz IS NULL OR issued_at >= $1)
				  AND ($2::timestamptz IS NULL OR issued_at < $2::timestamptz + INTERVAL '1 day')),
			(SELECT COUNT(*) FROM entitlements
				WHERE status = 'used'
				  AND ($1::timestamptz IS NULL OR used_at >= $1)
				  AND ($2::timestamptz IS NULL OR used_at < $2::timestamptz + INTERVAL '1 day')),
			(SELECT COUNT(DISTINCT student_id) FROM orders
				WHERE status = 'paid'
				  AND ($1::timestamptz IS NULL OR paid_at >= $1)
				  AND ($2::timestamptz IS NULL OR paid_at < $2::timestamptz + INTERVAL '1 day'))
	`, from, to).Scan(&s.TotalRevenueCents, &s.PaidOrders, &s.EntitlementsIssued, &s.EntitlementsUsed, &s.ActiveStudents)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

type SalesByProduct struct {
	ProductTitle string `json:"productTitle"`
	QuantitySold int    `json:"quantitySold"`
	RevenueCents int    `json:"revenueCents"`
}

// SalesByProduct returns every active product — including ones with zero sales so far —
// with how many units were sold (one paid order = one unit of the product it bought) and
// the revenue that product generated. from/to bound o.paid_at inside the inner subquery
// (not a WHERE on the outer join), so a product with no sales in range still lists with
// zero quantity/revenue instead of disappearing.
func (r *DashboardRepository) SalesByProduct(ctx context.Context, from, to *time.Time) ([]SalesByProduct, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT p.title, COALESCE(sub.quantity, 0), COALESCE(sub.revenue_cents, 0)
		FROM products p
		LEFT JOIN (
			SELECT ordered.product_id, COUNT(*) AS quantity, SUM(ordered.total_cents) AS revenue_cents
			FROM (
				SELECT DISTINCT o.id, o.total_cents, oi.product_id
				FROM orders o
				JOIN order_items oi ON oi.order_id = o.id
				WHERE o.status = 'paid'
				  AND ($1::timestamptz IS NULL OR o.paid_at >= $1)
				  AND ($2::timestamptz IS NULL OR o.paid_at < $2::timestamptz + INTERVAL '1 day')
			) ordered
			GROUP BY ordered.product_id
		) sub ON sub.product_id = p.id
		WHERE p.active = true
		ORDER BY 3 DESC, p.display_order
	`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]SalesByProduct, 0)
	for rows.Next() {
		var s SalesByProduct
		if err := rows.Scan(&s.ProductTitle, &s.QuantitySold, &s.RevenueCents); err != nil {
			return nil, err
		}
		list = append(list, s)
	}
	return list, rows.Err()
}

type SalesPoint struct {
	Day          string `json:"day"`
	RevenueCents int    `json:"revenueCents"`
	Orders       int    `json:"orders"`
}

func (r *DashboardRepository) SalesTimeseries(ctx context.Context, from, to time.Time) ([]SalesPoint, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT to_char(d::date, 'YYYY-MM-DD'),
		       COALESCE(SUM(o.total_cents), 0),
		       COUNT(o.id)
		FROM generate_series($1::date, $2::date, interval '1 day') AS d
		LEFT JOIN orders o ON o.paid_at::date = d::date AND o.status = 'paid'
		GROUP BY d
		ORDER BY d
	`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]SalesPoint, 0)
	for rows.Next() {
		var p SalesPoint
		if err := rows.Scan(&p.Day, &p.RevenueCents, &p.Orders); err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	return list, rows.Err()
}
