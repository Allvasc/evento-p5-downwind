package postgres

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Fixed seed UUIDs from migration 00003_vendor_scoped_entitlements.sql — stable across
// environments, safe to hardcode in tests instead of looking them up.
const (
	vendorAYO = "00000000-0000-0000-0000-0000000000a1"
	vendorP5  = "00000000-0000-0000-0000-0000000000f5"
)

// Every helper below inserts one row needed to exercise a repository method under test
// and registers a best-effort cleanup so integration tests don't leave fixture data
// behind in a shared dev database. They insert directly via SQL (not through other
// repositories) to keep each test's fixture graph minimal and explicit.

func mustInsertStudent(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	var id string
	email := fmt.Sprintf("test-%s@p5wellness.test", uuid.NewString())
	err := pool.QueryRow(context.Background(), `
		INSERT INTO students (full_name, email, password_hash) VALUES ('Teste Concorrência', $1, 'x')
		RETURNING id
	`, email).Scan(&id)
	if err != nil {
		t.Fatalf("insert test student: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM students WHERE id = $1`, id) })
	return id
}

func mustInsertActivity(t *testing.T, pool *pgxpool.Pool, vendorID string) string {
	t.Helper()
	var id string
	slug := "test-activity-" + uuid.NewString()[:8]
	err := pool.QueryRow(context.Background(), `
		INSERT INTO activities (title, slug, vendor_id) VALUES ('Atividade Teste', $1, $2)
		RETURNING id
	`, slug, vendorID).Scan(&id)
	if err != nil {
		t.Fatalf("insert test activity: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM activities WHERE id = $1`, id) })
	return id
}

func mustInsertSession(t *testing.T, pool *pgxpool.Pool, activityID string, capacity int) string {
	t.Helper()
	var id string
	starts := time.Now().Add(24 * time.Hour)
	ends := starts.Add(time.Hour)
	err := pool.QueryRow(context.Background(), `
		INSERT INTO class_sessions (activity_id, starts_at, ends_at, capacity)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, activityID, starts, ends, capacity).Scan(&id)
	if err != nil {
		t.Fatalf("insert test session: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM class_sessions WHERE id = $1`, id) })
	return id
}

func mustInsertProduct(t *testing.T, pool *pgxpool.Pool, activityID string) string {
	t.Helper()
	var id string
	slug := "test-product-" + uuid.NewString()[:8]
	err := pool.QueryRow(context.Background(), `
		INSERT INTO products (title, slug, type, price_cents) VALUES ('Produto Teste', $1, 'class', 1000)
		RETURNING id
	`, slug).Scan(&id)
	if err != nil {
		t.Fatalf("insert test product: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM products WHERE id = $1`, id) })

	if _, err := pool.Exec(context.Background(), `
		INSERT INTO product_activities (product_id, activity_id) VALUES ($1, $2)
	`, id, activityID); err != nil {
		t.Fatalf("link test product to activity: %v", err)
	}
	return id
}

func mustInsertPaidOrder(t *testing.T, pool *pgxpool.Pool, studentID string) string {
	t.Helper()
	var id string
	orderNumber := "TEST-" + uuid.NewString()[:8]
	err := pool.QueryRow(context.Background(), `
		INSERT INTO orders (order_number, student_id, status, total_cents, paid_at)
		VALUES ($1, $2, 'paid', 1000, now())
		RETURNING id
	`, orderNumber, studentID).Scan(&id)
	if err != nil {
		t.Fatalf("insert test order: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM orders WHERE id = $1`, id) })
	return id
}

func mustInsertOrderItem(t *testing.T, pool *pgxpool.Pool, orderID, activityID, sessionID string) string {
	t.Helper()
	productID := mustInsertProduct(t, pool, activityID)
	var id string
	err := pool.QueryRow(context.Background(), `
		INSERT INTO order_items (order_id, product_id, activity_id, class_session_id, benefit_type, unit_price_cents)
		VALUES ($1, $2, $3, $4, 'class', 1000)
		RETURNING id
	`, orderID, productID, activityID, sessionID).Scan(&id)
	if err != nil {
		t.Fatalf("insert test order item: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM order_items WHERE id = $1`, id) })
	return id
}

func mustInsertTeamMember(t *testing.T, pool *pgxpool.Pool, role, vendorID string) string {
	t.Helper()
	var id string
	email := fmt.Sprintf("test-staff-%s@p5wellness.test", uuid.NewString())
	err := pool.QueryRow(context.Background(), `
		INSERT INTO team_members (name, email, password_hash, role, vendor_id) VALUES ('Staff Teste', $1, 'x', $2, $3)
		RETURNING id
	`, email, role, vendorID).Scan(&id)
	if err != nil {
		t.Fatalf("insert test team member: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM team_members WHERE id = $1`, id) })
	return id
}

func mustInsertEntitlement(t *testing.T, pool *pgxpool.Pool, orderID, studentID, vendorID, token, orderItemID string) string {
	t.Helper()
	var id string
	today := time.Now().Format("2006-01-02")
	err := pool.QueryRow(context.Background(), `
		INSERT INTO entitlements (order_id, student_id, vendor_id, qr_token, status, valid_from, valid_until)
		VALUES ($1, $2, $3, $4, 'available', $5::date, $5::date)
		RETURNING id
	`, orderID, studentID, vendorID, token, today).Scan(&id)
	if err != nil {
		t.Fatalf("insert test entitlement: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM validation_log WHERE entitlement_id = $1`, id)
		_, _ = pool.Exec(context.Background(), `DELETE FROM entitlement_items WHERE entitlement_id = $1`, id)
		_, _ = pool.Exec(context.Background(), `DELETE FROM entitlements WHERE id = $1`, id)
	})

	if _, err := pool.Exec(context.Background(), `
		INSERT INTO entitlement_items (entitlement_id, order_item_id) VALUES ($1, $2)
	`, id, orderItemID); err != nil {
		t.Fatalf("link test entitlement to order item: %v", err)
	}
	return id
}
