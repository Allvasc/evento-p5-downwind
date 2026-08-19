package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// mustInsertSessionOnDay inserts a scheduled session landing on a specific Fortaleza
// calendar day — mustInsertSession only offers "24h from now", which isn't controllable
// enough to test day-based rescheduling.
func mustInsertSessionOnDay(t *testing.T, pool *pgxpool.Pool, activityID, day string, capacity int) string {
	t.Helper()
	starts, err := time.Parse("2006-01-02T15:04:05", day+"T15:00:00") // midday in Fortaleza (UTC-3), well clear of the date boundary
	if err != nil {
		t.Fatalf("parse day: %v", err)
	}
	var id string
	err = pool.QueryRow(context.Background(), `
		INSERT INTO class_sessions (activity_id, starts_at, ends_at, capacity)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, activityID, starts, starts.Add(time.Hour), capacity).Scan(&id)
	if err != nil {
		t.Fatalf("insert test session: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM class_sessions WHERE id = $1`, id) })
	return id
}

func futureDay(offsetDays int) string {
	return time.Now().AddDate(0, 0, offsetDays).Format("2006-01-02")
}

// TestReschedule_MovesWholeOrderToNewDay covers the "cliente não compareceu, quer outra
// data" flow end to end: a paid combo order (Yoga + HYROX, one shared AYO entitlement)
// moves to a day that has matching turmas for both activities, and every entitlement's
// validity window follows along — including one with no session of its own (the café
// da manhã ticket), which should still pick up the new day.
func TestReschedule_MovesWholeOrderToNewDay(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	repo := NewOrderRepository(pool)

	oldDay := futureDay(3)
	newDay := futureDay(10)

	yogaID := mustInsertActivity(t, pool, vendorAYO)
	hyroxID := mustInsertActivity(t, pool, vendorAYO)
	oldYogaSession := mustInsertSessionOnDay(t, pool, yogaID, oldDay, 10)
	oldHyroxSession := mustInsertSessionOnDay(t, pool, hyroxID, oldDay, 10)
	newYogaSession := mustInsertSessionOnDay(t, pool, yogaID, newDay, 10)
	newHyroxSession := mustInsertSessionOnDay(t, pool, hyroxID, newDay, 10)

	studentID := mustInsertStudent(t, pool)
	teamMemberID := mustInsertTeamMember(t, pool, "admin", vendorAYO)
	orderID := mustInsertPaidOrder(t, pool, studentID)
	yogaItemID := mustInsertOrderItem(t, pool, orderID, yogaID, oldYogaSession)
	hyroxItemID := mustInsertOrderItem(t, pool, orderID, hyroxID, oldHyroxSession)

	classEntitlementID := mustInsertEntitlement(t, pool, orderID, studentID, vendorAYO, "test-"+uuid.NewString(), yogaItemID)
	if _, err := pool.Exec(ctx, `INSERT INTO entitlement_items (entitlement_id, order_item_id) VALUES ($1, $2)`, classEntitlementID, hyroxItemID); err != nil {
		t.Fatalf("link hyrox item to entitlement: %v", err)
	}

	// Breakfast: no activity, no session of its own — its entitlement's date is still
	// expected to move to match the reschedule.
	breakfastProductID := mustInsertProduct(t, pool, yogaID)
	var breakfastItemID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO order_items (order_id, product_id, activity_id, benefit_type, unit_price_cents)
		VALUES ($1, $2, NULL, 'breakfast', 0) RETURNING id
	`, orderID, breakfastProductID).Scan(&breakfastItemID); err != nil {
		t.Fatalf("insert breakfast item: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM order_items WHERE id = $1`, breakfastItemID) })
	breakfastEntitlementID := mustInsertEntitlement(t, pool, orderID, studentID, vendorP5, "test-"+uuid.NewString(), breakfastItemID)

	result, err := repo.Reschedule(ctx, orderID, newDay, "cliente não compareceu, remarcado a pedido dele", teamMemberID)
	if err != nil {
		t.Fatalf("Reschedule: %v", err)
	}
	if result.NewDate != newDay {
		t.Errorf("result.NewDate = %s, want %s", result.NewDate, newDay)
	}
	// mustInsertEntitlement always stamps valid_from as today regardless of the linked
	// session's day, so that — not oldDay — is what Reschedule should report as the
	// previous date (it reads the entitlement's own valid_from, not the old session).
	if result.PreviousDate != futureDay(0) {
		t.Errorf("result.PreviousDate = %s, want %s (today, per mustInsertEntitlement)", result.PreviousDate, futureDay(0))
	}

	var gotYogaSession, gotHyroxSession string
	if err := pool.QueryRow(ctx, `SELECT class_session_id FROM order_items WHERE id = $1`, yogaItemID).Scan(&gotYogaSession); err != nil {
		t.Fatalf("read yoga item: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT class_session_id FROM order_items WHERE id = $1`, hyroxItemID).Scan(&gotHyroxSession); err != nil {
		t.Fatalf("read hyrox item: %v", err)
	}
	if gotYogaSession != newYogaSession {
		t.Errorf("yoga order_item.class_session_id = %s, want %s (new session)", gotYogaSession, newYogaSession)
	}
	if gotHyroxSession != newHyroxSession {
		t.Errorf("hyrox order_item.class_session_id = %s, want %s (new session)", gotHyroxSession, newHyroxSession)
	}

	for _, entID := range []string{classEntitlementID, breakfastEntitlementID} {
		var validFrom, validUntil string
		if err := pool.QueryRow(ctx, `SELECT valid_from::text, valid_until::text FROM entitlements WHERE id = $1`, entID).Scan(&validFrom, &validUntil); err != nil {
			t.Fatalf("read entitlement %s: %v", entID, err)
		}
		if validFrom != newDay || validUntil != newDay {
			t.Errorf("entitlement %s valid_from/valid_until = %s/%s, want both %s", entID, validFrom, validUntil, newDay)
		}
	}

	var reason, changedBy string
	if err := pool.QueryRow(ctx, `SELECT reason, changed_by FROM order_reschedules WHERE order_id = $1`, orderID).Scan(&reason, &changedBy); err != nil {
		t.Fatalf("read audit row: %v", err)
	}
	if changedBy != teamMemberID {
		t.Errorf("audit changed_by = %s, want %s", changedBy, teamMemberID)
	}
	if reason == "" {
		t.Error("audit reason should not be empty")
	}

	// The old sessions must have their seats freed up now that the order moved away.
	oldYogaHasCapacity, err := (&SessionRepository{pool: pool}).HasCapacity(ctx, oldYogaSession)
	if err != nil {
		t.Fatalf("HasCapacity old yoga: %v", err)
	}
	if !oldYogaHasCapacity {
		t.Error("old yoga session should have its seat freed after reschedule")
	}
}

// TestReschedule_NoMatchingSessionOnDate confirms a clear error instead of a silent
// partial move when the target day has no turma for one of the order's activities.
func TestReschedule_NoMatchingSessionOnDate(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	repo := NewOrderRepository(pool)

	oldDay := futureDay(3)
	yogaID := mustInsertActivity(t, pool, vendorAYO)
	oldSession := mustInsertSessionOnDay(t, pool, yogaID, oldDay, 10)

	studentID := mustInsertStudent(t, pool)
	teamMemberID := mustInsertTeamMember(t, pool, "admin", vendorAYO)
	orderID := mustInsertPaidOrder(t, pool, studentID)
	itemID := mustInsertOrderItem(t, pool, orderID, yogaID, oldSession)
	mustInsertEntitlement(t, pool, orderID, studentID, vendorAYO, "test-"+uuid.NewString(), itemID)

	_, err := repo.Reschedule(ctx, orderID, futureDay(20), "sem turma nesse dia", teamMemberID)
	if !errors.Is(err, ErrNoSessionOnDate) {
		t.Fatalf("got error %v, want ErrNoSessionOnDate", err)
	}

	// Nothing should have moved.
	var gotSession string
	if err := pool.QueryRow(ctx, `SELECT class_session_id FROM order_items WHERE id = $1`, itemID).Scan(&gotSession); err != nil {
		t.Fatalf("read item: %v", err)
	}
	if gotSession != oldSession {
		t.Errorf("order_item.class_session_id changed to %s despite the error, want unchanged %s", gotSession, oldSession)
	}
}

// TestReschedule_BlocksWhenTicketAlreadyUsed protects against moving a ticket for a visit
// that already happened.
func TestReschedule_BlocksWhenTicketAlreadyUsed(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	repo := NewOrderRepository(pool)

	oldDay := futureDay(3)
	yogaID := mustInsertActivity(t, pool, vendorAYO)
	oldSession := mustInsertSessionOnDay(t, pool, yogaID, oldDay, 10)
	mustInsertSessionOnDay(t, pool, yogaID, futureDay(20), 10)

	studentID := mustInsertStudent(t, pool)
	teamMemberID := mustInsertTeamMember(t, pool, "admin", vendorAYO)
	orderID := mustInsertPaidOrder(t, pool, studentID)
	itemID := mustInsertOrderItem(t, pool, orderID, yogaID, oldSession)
	entitlementID := mustInsertEntitlement(t, pool, orderID, studentID, vendorAYO, "test-"+uuid.NewString(), itemID)

	if _, err := pool.Exec(ctx, `UPDATE entitlements SET status = 'used', used_at = now(), used_by = $1 WHERE id = $2`, teamMemberID, entitlementID); err != nil {
		t.Fatalf("mark entitlement used: %v", err)
	}

	_, err := repo.Reschedule(ctx, orderID, futureDay(20), "tentativa depois de já ter usado", teamMemberID)
	if !errors.Is(err, ErrTicketAlreadyUsed) {
		t.Fatalf("got error %v, want ErrTicketAlreadyUsed", err)
	}
}
