package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AdminCatalogRepository struct {
	pool *pgxpool.Pool
}

func NewAdminCatalogRepository(pool *pgxpool.Pool) *AdminCatalogRepository {
	return &AdminCatalogRepository{pool: pool}
}

type AdminActivity struct {
	ID              string `json:"id"`
	Title           string `json:"title"`
	Slug            string `json:"slug"`
	Instructor      string `json:"instructor"`
	DurationMinutes int    `json:"durationMinutes"`
	Description     string `json:"description"`
	Active          bool   `json:"active"`
	VendorID        string `json:"vendorId"`
	VendorName      string `json:"vendorName"`
}

func (r *AdminCatalogRepository) ListActivities(ctx context.Context) ([]AdminActivity, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT a.id, a.title, a.slug, COALESCE(a.instructor,''), COALESCE(a.duration_minutes,0), COALESCE(a.description,''), a.active,
		       a.vendor_id, v.name
		FROM activities a
		JOIN vendors v ON v.id = a.vendor_id
		ORDER BY a.display_order, a.created_at
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]AdminActivity, 0)
	for rows.Next() {
		var a AdminActivity
		if err := rows.Scan(&a.ID, &a.Title, &a.Slug, &a.Instructor, &a.DurationMinutes, &a.Description, &a.Active, &a.VendorID, &a.VendorName); err != nil {
			return nil, err
		}
		list = append(list, a)
	}
	return list, rows.Err()
}

func (r *AdminCatalogRepository) CreateActivity(ctx context.Context, title, instructor string, durationMinutes int, description, vendorID string) (string, error) {
	slug := slugify(title)
	var id string
	err := r.pool.QueryRow(ctx, `
		INSERT INTO activities (title, slug, instructor, duration_minutes, description, vendor_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, title, slug, nullIfEmpty(instructor), nullIfZero(durationMinutes), nullIfEmpty(description), vendorID).Scan(&id)
	return id, err
}

func (r *AdminCatalogRepository) SetActivityActive(ctx context.Context, id string, active bool) error {
	_, err := r.pool.Exec(ctx, `UPDATE activities SET active = $1, updated_at = now() WHERE id = $2`, active, id)
	return err
}

func (r *AdminCatalogRepository) UpdateActivity(ctx context.Context, id, title, instructor string, durationMinutes int, description, vendorID string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE activities
		SET title = $1, instructor = $2, duration_minutes = $3, description = $4, vendor_id = $5, updated_at = now()
		WHERE id = $6
	`, title, nullIfEmpty(instructor), nullIfZero(durationMinutes), nullIfEmpty(description), vendorID, id)
	return err
}

type AdminProduct struct {
	ID                string   `json:"id"`
	Title             string   `json:"title"`
	Slug              string   `json:"slug"`
	Description       string   `json:"description"`
	Type              string   `json:"type"`
	IncludesBreakfast bool     `json:"includesBreakfast"`
	PriceCents        int      `json:"priceCents"`
	Featured          bool     `json:"featured"`
	Active            bool     `json:"active"`
	ChooseOneActivity bool     `json:"chooseOneActivity"`
	ActivityIDs       []string `json:"activityIds"`
}

func (r *AdminCatalogRepository) ListProducts(ctx context.Context) ([]AdminProduct, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, title, slug, COALESCE(description,''), type, includes_breakfast, price_cents, featured, active, choose_one_activity
		FROM products ORDER BY display_order, created_at
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]AdminProduct, 0)
	index := map[string]int{}
	for rows.Next() {
		var p AdminProduct
		if err := rows.Scan(&p.ID, &p.Title, &p.Slug, &p.Description, &p.Type, &p.IncludesBreakfast, &p.PriceCents, &p.Featured, &p.Active, &p.ChooseOneActivity); err != nil {
			return nil, err
		}
		p.ActivityIDs = []string{}
		index[p.ID] = len(list)
		list = append(list, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	linkRows, err := r.pool.Query(ctx, `SELECT product_id, activity_id FROM product_activities`)
	if err != nil {
		return nil, err
	}
	defer linkRows.Close()
	for linkRows.Next() {
		var productID, activityID string
		if err := linkRows.Scan(&productID, &activityID); err != nil {
			return nil, err
		}
		if i, ok := index[productID]; ok {
			list[i].ActivityIDs = append(list[i].ActivityIDs, activityID)
		}
	}

	return list, linkRows.Err()
}

type CreateProductParams struct {
	Title             string
	Description       string
	Type              string
	IncludesBreakfast bool
	PriceCents        int
	Featured          bool
	ChooseOneActivity bool
	ActivityIDs       []string
}

func (r *AdminCatalogRepository) CreateProduct(ctx context.Context, p CreateProductParams) (string, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	slug := slugify(p.Title)
	var id string
	err = tx.QueryRow(ctx, `
		INSERT INTO products (title, slug, description, type, includes_breakfast, price_cents, featured, choose_one_activity)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`, p.Title, slug, nullIfEmpty(p.Description), p.Type, p.IncludesBreakfast, p.PriceCents, p.Featured, p.ChooseOneActivity).Scan(&id)
	if err != nil {
		return "", err
	}

	for _, activityID := range p.ActivityIDs {
		if _, err := tx.Exec(ctx, `INSERT INTO product_activities (product_id, activity_id) VALUES ($1, $2)`, id, activityID); err != nil {
			return "", err
		}
	}

	return id, tx.Commit(ctx)
}

func (r *AdminCatalogRepository) SetProductActive(ctx context.Context, id string, active bool) error {
	_, err := r.pool.Exec(ctx, `UPDATE products SET active = $1, updated_at = now() WHERE id = $2`, active, id)
	return err
}

// UpdateProduct rewrites a product's fields and fully replaces its activity links
// (simpler and less error-prone than diffing old vs. new for a small admin-only list).
func (r *AdminCatalogRepository) UpdateProduct(ctx context.Context, id string, p CreateProductParams) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
		UPDATE products
		SET title = $1, description = $2, type = $3, includes_breakfast = $4, price_cents = $5, featured = $6, choose_one_activity = $7, updated_at = now()
		WHERE id = $8
	`, p.Title, nullIfEmpty(p.Description), p.Type, p.IncludesBreakfast, p.PriceCents, p.Featured, p.ChooseOneActivity, id); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `DELETE FROM product_activities WHERE product_id = $1`, id); err != nil {
		return err
	}
	for _, activityID := range p.ActivityIDs {
		if _, err := tx.Exec(ctx, `INSERT INTO product_activities (product_id, activity_id) VALUES ($1, $2)`, id, activityID); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}
