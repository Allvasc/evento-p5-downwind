package postgres

import (
	"context"
	"errors"
	"sync"
	"testing"

	"p5wellness/backend/internal/domain/product"
)

// TestCreateOrder_ConcurrentBookingsRespectCapacity is the checkout-side counterpart of
// the check-in race test: a session with 1 open seat and two students racing to book it
// must produce exactly one order, not two overbooking the same class. CreateOrder relies
// on `SELECT ... FOR UPDATE` inside the transaction to serialize the capacity check —
// this test fails if that locking ever regresses.
func TestCreateOrder_ConcurrentBookingsRespectCapacity(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	repo := NewOrderRepository(pool)

	activityID := mustInsertActivity(t, pool, vendorAYO)
	sessionID := mustInsertSession(t, pool, activityID, 1) // exactly 1 seat

	const contenders = 5
	studentIDs := make([]string, contenders)
	for i := range studentIDs {
		studentIDs[i] = mustInsertStudent(t, pool)
	}

	prod := product.Product{
		ID:         "unused", // CreateOrder only uses this to stamp order_items.product_id
		Title:      "Atividade Teste",
		Type:       product.TypeClass,
		PriceCents: 1000,
		Activities: []product.Activity{{ID: activityID, Title: "Atividade Teste"}},
	}
	// order_items.product_id has a FK — point it at a real product referencing the activity.
	productID := mustInsertProduct(t, pool, activityID)
	prod.ID = productID

	results := make([]error, contenders)
	orderIDs := make([]string, contenders)
	var wg sync.WaitGroup
	for i := 0; i < contenders; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			order, err := repo.CreateOrder(ctx, studentIDs[i], prod, map[string]string{activityID: sessionID})
			results[i] = err
			if order != nil {
				orderIDs[i] = order.ID
			}
		}(i)
	}
	wg.Wait()

	succeeded, full := 0, 0
	for _, err := range results {
		switch {
		case err == nil:
			succeeded++
		case errors.Is(err, ErrSessionFull):
			full++
		default:
			t.Errorf("unexpected error: %v", err)
		}
	}
	t.Cleanup(func() {
		for _, id := range orderIDs {
			if id != "" {
				_, _ = pool.Exec(context.Background(), `DELETE FROM order_items WHERE order_id = $1`, id)
				_, _ = pool.Exec(context.Background(), `DELETE FROM orders WHERE id = $1`, id)
			}
		}
	})

	if succeeded != 1 {
		t.Errorf("got %d successful bookings for a 1-seat session, want exactly 1 (got %d rejected as full)", succeeded, full)
	}
	if full != contenders-1 {
		t.Errorf("got %d rejected as full, want %d", full, contenders-1)
	}
}
