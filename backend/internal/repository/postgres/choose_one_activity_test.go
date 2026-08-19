package postgres

import (
	"context"
	"errors"
	"testing"

	"p5wellness/backend/internal/domain/product"
)

// TestCreateOrder_ChooseOneActivity locks in the "Yoga ou HYROX" behavior: a product
// with ChooseOneActivity=true and both activities linked must book only the one the
// customer actually selected, not both.
func TestCreateOrder_ChooseOneActivity(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	repo := NewOrderRepository(pool)

	yogaID := mustInsertActivity(t, pool, vendorAYO)
	hyroxID := mustInsertActivity(t, pool, vendorAYO)
	yogaSession := mustInsertSession(t, pool, yogaID, 10)
	hyroxSession := mustInsertSession(t, pool, hyroxID, 10)
	studentID := mustInsertStudent(t, pool)
	productID := mustInsertProduct(t, pool, yogaID)
	if _, err := pool.Exec(ctx, `INSERT INTO product_activities (product_id, activity_id) VALUES ($1, $2)`, productID, hyroxID); err != nil {
		t.Fatalf("link second activity: %v", err)
	}

	p := product.Product{
		ID:                productID,
		Title:             "Aula Individual",
		Type:              product.TypeClass,
		PriceCents:        2500,
		ChooseOneActivity: true,
		Activities: []product.Activity{
			{ID: yogaID, Title: "Yoga"},
			{ID: hyroxID, Title: "HYROX"},
		},
	}

	order, err := repo.CreateOrder(ctx, studentID, p, map[string]string{hyroxID: hyroxSession})
	if err != nil {
		t.Fatalf("CreateOrder: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM order_items WHERE order_id = $1`, order.ID)
		_, _ = pool.Exec(ctx, `DELETE FROM orders WHERE id = $1`, order.ID)
	})

	var count int
	var bookedActivityID string
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM order_items WHERE order_id = $1`, order.ID).Scan(&count); err != nil {
		t.Fatalf("count order_items: %v", err)
	}
	if count != 1 {
		t.Fatalf("got %d order_items, want exactly 1 (chose HYROX only, Yoga should not be booked)", count)
	}
	if err := pool.QueryRow(ctx, `SELECT activity_id FROM order_items WHERE order_id = $1`, order.ID).Scan(&bookedActivityID); err != nil {
		t.Fatalf("read booked activity: %v", err)
	}
	if bookedActivityID != hyroxID {
		t.Errorf("booked activity = %s, want %s (HYROX)", bookedActivityID, hyroxID)
	}

	// Yoga's own session must remain untouched — booking HYROX must not also consume a
	// seat on the activity that wasn't chosen.
	hasCapacity, err := (&SessionRepository{pool: pool}).HasCapacity(ctx, yogaSession)
	if err != nil {
		t.Fatalf("HasCapacity: %v", err)
	}
	if !hasCapacity {
		t.Error("Yoga session capacity was consumed even though HYROX was the chosen activity")
	}
}

// TestCreateOrder_ChooseOneActivity_NoneSelected confirms a clean, specific error when
// the customer submits neither activity's session.
func TestCreateOrder_ChooseOneActivity_NoneSelected(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	repo := NewOrderRepository(pool)

	yogaID := mustInsertActivity(t, pool, vendorAYO)
	hyroxID := mustInsertActivity(t, pool, vendorAYO)
	mustInsertSession(t, pool, yogaID, 10)
	studentID := mustInsertStudent(t, pool)
	productID := mustInsertProduct(t, pool, yogaID)
	if _, err := pool.Exec(ctx, `INSERT INTO product_activities (product_id, activity_id) VALUES ($1, $2)`, productID, hyroxID); err != nil {
		t.Fatalf("link second activity: %v", err)
	}

	p := product.Product{
		ID:                productID,
		Type:              product.TypeClass,
		PriceCents:        2500,
		ChooseOneActivity: true,
		Activities: []product.Activity{
			{ID: yogaID, Title: "Yoga"},
			{ID: hyroxID, Title: "HYROX"},
		},
	}

	_, err := repo.CreateOrder(ctx, studentID, p, map[string]string{})
	if !errors.Is(err, ErrActivityChoiceRequired) {
		t.Fatalf("got error %v, want ErrActivityChoiceRequired", err)
	}
}
