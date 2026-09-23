package jobs

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"p5wellness/backend/internal/asaas"
	"p5wellness/backend/internal/repository/postgres"
)

// Integration test against a real Postgres, skipped when DATABASE_URL isn't set — same
// convention as internal/repository/postgres/testutil_test.go.
func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	_ = godotenv.Load("../../.env")
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set — skipping integration test (see backend/.env)")
	}
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatalf("connect to test database: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func mustExec(t *testing.T, pool *pgxpool.Pool, sql string, args ...any) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), sql, args...); err != nil {
		t.Fatalf("exec %q: %v", sql, err)
	}
}

// fixture: one session with capacity 1 and a product/activity to book it.
func mustSession(t *testing.T, pool *pgxpool.Pool) (sessionID, activityID, productID string) {
	t.Helper()
	ctx := context.Background()
	suffix := uuid.NewString()[:8]
	if err := pool.QueryRow(ctx, `INSERT INTO activities (title, slug, vendor_id) VALUES ('Expirer Teste', $1, '00000000-0000-0000-0000-0000000000f5') RETURNING id`,
		"test-exp-act-"+suffix).Scan(&activityID); err != nil {
		t.Fatalf("insert activity: %v", err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO products (title, slug, type, price_cents) VALUES ('Expirer Teste', $1, 'class', 1000) RETURNING id`,
		"test-exp-prod-"+suffix).Scan(&productID); err != nil {
		t.Fatalf("insert product: %v", err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO class_sessions (activity_id, starts_at, ends_at, capacity) VALUES ($1, now() + interval '1 day', now() + interval '1 day 1 hour', 1) RETURNING id`,
		activityID).Scan(&sessionID); err != nil {
		t.Fatalf("insert session: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM class_sessions WHERE id = $1`, sessionID)
		_, _ = pool.Exec(ctx, `DELETE FROM products WHERE id = $1`, productID)
		_, _ = pool.Exec(ctx, `DELETE FROM activities WHERE id = $1`, activityID)
	})
	return
}

// mustPendingOrder books the session with a pending order created `age` ago, tied to paymentID.
func mustPendingOrder(t *testing.T, pool *pgxpool.Pool, sessionID, activityID, productID, paymentID string, age time.Duration) string {
	t.Helper()
	ctx := context.Background()
	var studentID, orderID string
	if err := pool.QueryRow(ctx, `INSERT INTO students (full_name, email, password_hash) VALUES ('Expirer Teste', $1, 'x') RETURNING id`,
		fmt.Sprintf("test-exp-%s@p5wellness.test", uuid.NewString())).Scan(&studentID); err != nil {
		t.Fatalf("insert student: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO orders (order_number, student_id, status, total_cents, payment_method, asaas_payment_id, created_at)
		VALUES ($1, $2, 'pending', 1000, 'pix', NULLIF($3, ''), now() - $4::interval) RETURNING id`,
		"TEST-EXP-"+uuid.NewString()[:8], studentID, paymentID, age.String()).Scan(&orderID); err != nil {
		t.Fatalf("insert order: %v", err)
	}
	mustExec(t, pool, `INSERT INTO order_items (order_id, product_id, activity_id, class_session_id, benefit_type, unit_price_cents) VALUES ($1, $2, $3, $4, 'class', 1000)`,
		orderID, productID, activityID, sessionID)
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM orders WHERE id = $1`, orderID)
		_, _ = pool.Exec(ctx, `DELETE FROM students WHERE id = $1`, studentID)
	})
	return orderID
}

func orderStatus(t *testing.T, pool *pgxpool.Pool, orderID string) string {
	t.Helper()
	var s string
	if err := pool.QueryRow(context.Background(), `SELECT status FROM orders WHERE id = $1`, orderID).Scan(&s); err != nil {
		t.Fatalf("read order status: %v", err)
	}
	return s
}

func openSeats(t *testing.T, pool *pgxpool.Pool, activityID string) int {
	t.Helper()
	list, err := postgres.NewSessionRepository(pool).AvailableForActivity(context.Background(), activityID)
	if err != nil {
		t.Fatalf("available sessions: %v", err)
	}
	return len(list)
}

func newCharge(t *testing.T, fake *asaas.FakeClient) string {
	t.Helper()
	c, err := fake.CreateCharge(context.Background(), asaas.ChargeParams{ValueCents: 1000})
	if err != nil {
		t.Fatalf("fake charge: %v", err)
	}
	return c.PaymentID
}

func TestExpirer_UnpaidPixAfter10MinReleasesSeat(t *testing.T) {
	pool := testPool(t)
	orders := postgres.NewOrderRepository(pool)
	fake := asaas.NewFakeClient()
	expirer := NewPendingOrderExpirer(orders, fake, slog.New(slog.NewTextHandler(io.Discard, nil)))
	ctx := context.Background()

	sessionID, activityID, productID := mustSession(t, pool)
	paymentID := newCharge(t, fake)
	orderID := mustPendingOrder(t, pool, sessionID, activityID, productID, paymentID, 11*time.Minute)

	if openSeats(t, pool, activityID) != 0 {
		t.Fatal("pending order should be holding the only seat")
	}
	if _, left, _ := orders.PaymentStatus(ctx, orderID); left != 0 {
		t.Fatalf("secondsLeft = %d, want 0 for an 11-minute-old order", left)
	}

	expirer.Sweep(ctx)

	if got := orderStatus(t, pool, orderID); got != "expired" {
		t.Fatalf("order status = %q, want expired", got)
	}
	if p, _ := fake.GetPayment(ctx, paymentID); !p.Deleted {
		t.Fatalf("asaas charge should be deleted, got status %q", p.Status)
	}
	if openSeats(t, pool, activityID) != 1 {
		t.Fatal("seat should be free again after expiry")
	}
}

func TestExpirer_KeepsFreshAndPaidOrders(t *testing.T) {
	pool := testPool(t)
	orders := postgres.NewOrderRepository(pool)
	fake := asaas.NewFakeClient()
	expirer := NewPendingOrderExpirer(orders, fake, slog.New(slog.NewTextHandler(io.Discard, nil)))
	ctx := context.Background()

	// Fresh order (2 min old): untouched, ~8 min left on the countdown.
	s1, a1, p1 := mustSession(t, pool)
	fresh := mustPendingOrder(t, pool, s1, a1, p1, newCharge(t, fake), 2*time.Minute)

	// Stale order whose Pix was paid (webhook not processed yet): must keep its seat.
	s2, a2, p2 := mustSession(t, pool)
	paidPayment := newCharge(t, fake)
	fake.SimulateConfirmation(paidPayment)
	paid := mustPendingOrder(t, pool, s2, a2, p2, paidPayment, 15*time.Minute)

	expirer.Sweep(ctx)

	if got := orderStatus(t, pool, fresh); got != "pending" {
		t.Fatalf("fresh order status = %q, want pending", got)
	}
	if _, left, _ := orders.PaymentStatus(ctx, fresh); left < 470 || left > 480 {
		t.Fatalf("fresh order secondsLeft = %d, want ~480", left)
	}
	if got := orderStatus(t, pool, paid); got != "pending" {
		t.Fatalf("paid-at-Asaas order status = %q, want pending (webhook will confirm)", got)
	}
	if openSeats(t, pool, a2) != 0 {
		t.Fatal("paid order must keep its seat")
	}
}
