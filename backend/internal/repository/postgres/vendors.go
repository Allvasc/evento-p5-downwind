package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Vendor struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type VendorRepository struct {
	pool *pgxpool.Pool
}

func NewVendorRepository(pool *pgxpool.Pool) *VendorRepository {
	return &VendorRepository{pool: pool}
}

func (r *VendorRepository) List(ctx context.Context) ([]Vendor, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, name, slug FROM vendors ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]Vendor, 0)
	for rows.Next() {
		var v Vendor
		if err := rows.Scan(&v.ID, &v.Name, &v.Slug); err != nil {
			return nil, err
		}
		list = append(list, v)
	}
	return list, rows.Err()
}
