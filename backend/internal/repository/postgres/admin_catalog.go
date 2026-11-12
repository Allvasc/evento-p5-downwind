package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
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

// ListActivities lists the activities of one event ("" = every event).
func (r *AdminCatalogRepository) ListActivities(ctx context.Context, eventID string) ([]AdminActivity, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT a.id, a.title, a.slug, COALESCE(a.instructor,''), COALESCE(a.duration_minutes,0), COALESCE(a.description,''), a.active,
		       a.vendor_id, v.name
		FROM activities a
		JOIN vendors v ON v.id = a.vendor_id
		WHERE ($1::uuid IS NULL OR a.event_id = $1::uuid)
		ORDER BY a.display_order, a.created_at
	`, nullableEventID(eventID))
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

func (r *AdminCatalogRepository) CreateActivity(ctx context.Context, eventID, title, instructor string, durationMinutes int, description, vendorID string) (string, error) {
	slug := slugify(title)
	var id string
	err := r.pool.QueryRow(ctx, `
		INSERT INTO activities (title, slug, instructor, duration_minutes, description, vendor_id, event_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`, title, slug, nullIfEmpty(instructor), nullIfZero(durationMinutes), nullIfEmpty(description), vendorID, eventID).Scan(&id)
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

// ListProducts lists the products of one event ("" = every event).
func (r *AdminCatalogRepository) ListProducts(ctx context.Context, eventID string) ([]AdminProduct, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, title, slug, COALESCE(description,''), type, includes_breakfast, price_cents, featured, active, choose_one_activity
		FROM products
		WHERE ($1::uuid IS NULL OR event_id = $1::uuid)
		ORDER BY display_order, created_at
	`, nullableEventID(eventID))
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
	// EventID is only read by CreateProduct; an existing product never changes event.
	EventID           string
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

	if err := ensureActivitiesInEvent(ctx, tx, p.EventID, p.ActivityIDs); err != nil {
		return "", err
	}

	slug := slugify(p.Title)
	var id string
	err = tx.QueryRow(ctx, `
		INSERT INTO products (title, slug, description, type, includes_breakfast, price_cents, featured, choose_one_activity, event_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`, p.Title, slug, nullIfEmpty(p.Description), p.Type, p.IncludesBreakfast, p.PriceCents, p.Featured, p.ChooseOneActivity, p.EventID).Scan(&id)
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

	var productEventID string
	if err := tx.QueryRow(ctx, `SELECT event_id FROM products WHERE id = $1`, id).Scan(&productEventID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	if err := ensureActivitiesInEvent(ctx, tx, productEventID, p.ActivityIDs); err != nil {
		return err
	}

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

// ensureActivitiesInEvent refuses linking a product to an activity of another event —
// otherwise a ticket of event B could book a turma of event A.
func ensureActivitiesInEvent(ctx context.Context, tx pgx.Tx, eventID string, activityIDs []string) error {
	if len(activityIDs) == 0 {
		return nil
	}
	var outside int
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*) FROM activities WHERE id = ANY($1::uuid[]) AND event_id <> $2
	`, activityIDs, eventID).Scan(&outside); err != nil {
		return err
	}
	if outside > 0 {
		return ErrOtherEventActivity
	}
	return nil
}
