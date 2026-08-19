package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"p5wellness/backend/internal/domain/product"
)

type CatalogRepository struct {
	pool *pgxpool.Pool
}

func NewCatalogRepository(pool *pgxpool.Pool) *CatalogRepository {
	return &CatalogRepository{pool: pool}
}

// ListPublicProducts returns all active products, ordered for display, each with its
// linked activities (a combo can grant access to more than one activity).
func (r *CatalogRepository) GetProductByID(ctx context.Context, id string) (*product.Product, error) {
	var p product.Product
	err := r.pool.QueryRow(ctx, `
		SELECT id, title, slug, COALESCE(description, ''), type, includes_breakfast, price_cents, featured, choose_one_activity
		FROM products WHERE id = $1 AND active = true
	`, id).Scan(&p.ID, &p.Title, &p.Slug, &p.Description, &p.Type, &p.IncludesBreakfast, &p.PriceCents, &p.Featured, &p.ChooseOneActivity)
	if err != nil {
		return nil, err
	}

	rows, err := r.pool.Query(ctx, `
		SELECT a.id, a.title, a.slug, COALESCE(a.instructor, ''), COALESCE(a.duration_minutes, 0), COALESCE(a.description, '')
		FROM product_activities pa
		JOIN activities a ON a.id = pa.activity_id
		WHERE pa.product_id = $1 AND a.active = true
	`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	p.Activities = []product.Activity{}
	for rows.Next() {
		var a product.Activity
		if err := rows.Scan(&a.ID, &a.Title, &a.Slug, &a.Instructor, &a.DurationMinutes, &a.Description); err != nil {
			return nil, err
		}
		p.Activities = append(p.Activities, a)
	}
	return &p, rows.Err()
}

func (r *CatalogRepository) ListPublicProducts(ctx context.Context) ([]product.Product, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, title, slug, COALESCE(description, ''), type, includes_breakfast, price_cents, featured, choose_one_activity
		FROM products
		WHERE active = true
		ORDER BY display_order ASC, created_at ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	products := make([]product.Product, 0)
	index := map[string]int{}
	for rows.Next() {
		var p product.Product
		if err := rows.Scan(&p.ID, &p.Title, &p.Slug, &p.Description, &p.Type, &p.IncludesBreakfast, &p.PriceCents, &p.Featured, &p.ChooseOneActivity); err != nil {
			return nil, err
		}
		p.Activities = []product.Activity{}
		index[p.ID] = len(products)
		products = append(products, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(products) == 0 {
		return products, nil
	}

	actRows, err := r.pool.Query(ctx, `
		SELECT pa.product_id, a.id, a.title, a.slug, COALESCE(a.instructor, ''), COALESCE(a.duration_minutes, 0), COALESCE(a.description, '')
		FROM product_activities pa
		JOIN activities a ON a.id = pa.activity_id
		WHERE a.active = true
	`)
	if err != nil {
		return nil, err
	}
	defer actRows.Close()

	for actRows.Next() {
		var productID string
		var a product.Activity
		if err := actRows.Scan(&productID, &a.ID, &a.Title, &a.Slug, &a.Instructor, &a.DurationMinutes, &a.Description); err != nil {
			return nil, err
		}
		if i, ok := index[productID]; ok {
			products[i].Activities = append(products[i].Activities, a)
		}
	}
	if err := actRows.Err(); err != nil {
		return nil, err
	}

	return products, nil
}
