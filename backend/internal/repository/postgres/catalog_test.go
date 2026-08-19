package postgres

import (
	"context"
	"testing"
)

// TestListPublicProducts_NullDescription is a regression test for a real production
// incident: a product created directly via SQL (no admin-panel form, which always sends
// a description) left `description` NULL, and both ListPublicProducts and
// GetProductByID scanned it straight into a non-nullable Go string, crashing the entire
// public catalog with "não foi possível carregar o catálogo" until every row was
// patched. Both queries now COALESCE it to an empty string.
func TestListPublicProducts_NullDescription(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	repo := NewCatalogRepository(pool)

	activityID := mustInsertActivity(t, pool, vendorAYO)
	var productID string
	err := pool.QueryRow(ctx, `
		INSERT INTO products (title, slug, type, price_cents, active)
		VALUES ('Produto Sem Descrição', 'sem-descricao-test', 'class', 1000, true)
		RETURNING id
	`).Scan(&productID)
	if err != nil {
		t.Fatalf("insert test product: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(ctx, `DELETE FROM products WHERE id = $1`, productID) })
	if _, err := pool.Exec(ctx, `INSERT INTO product_activities (product_id, activity_id) VALUES ($1, $2)`, productID, activityID); err != nil {
		t.Fatalf("link test product: %v", err)
	}

	products, err := repo.ListPublicProducts(ctx)
	if err != nil {
		t.Fatalf("ListPublicProducts with a NULL description: %v", err)
	}
	found := false
	for _, p := range products {
		if p.ID == productID {
			found = true
			if p.Description != "" {
				t.Errorf("expected empty description, got %q", p.Description)
			}
		}
	}
	if !found {
		t.Error("test product not found in ListPublicProducts results")
	}

	got, err := repo.GetProductByID(ctx, productID)
	if err != nil {
		t.Fatalf("GetProductByID with a NULL description: %v", err)
	}
	if got.Description != "" {
		t.Errorf("expected empty description, got %q", got.Description)
	}
}
