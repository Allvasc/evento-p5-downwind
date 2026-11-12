package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"p5wellness/backend/internal/domain/product"
)

// These tests lock in the rule the multi-event feature exists for: two events on sale at
// the same time never mix — catalog, orders, vouchers, check-in and catalog links.

func mustInsertEvent(t *testing.T, pool *pgxpool.Pool, active bool) string {
	t.Helper()
	var id string
	slug := "evento-teste-" + uuid.NewString()[:8]
	err := pool.QueryRow(context.Background(), `
		INSERT INTO events (name, slug, active) VALUES ('Evento Teste', $1, $2) RETURNING id
	`, slug, active).Scan(&id)
	if err != nil {
		t.Fatalf("insert test event: %v", err)
	}
	// Registered first, so it runs last (t.Cleanup is LIFO) — after the rows pointing at it.
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM events WHERE id = $1`, id) })
	return id
}

// mustInsertEventCatalog creates an activity + session + active product inside eventID.
func mustInsertEventCatalog(t *testing.T, pool *pgxpool.Pool, eventID string) (activityID, sessionID, productID string) {
	t.Helper()
	ctx := context.Background()
	activityID = mustInsertActivity(t, pool, vendorP5)
	sessionID = mustInsertSession(t, pool, activityID, 10)
	productID = mustInsertProduct(t, pool, activityID)
	if _, err := pool.Exec(ctx, `UPDATE activities SET event_id = $1 WHERE id = $2`, eventID, activityID); err != nil {
		t.Fatalf("move activity to event: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE products SET event_id = $1 WHERE id = $2`, eventID, productID); err != nil {
		t.Fatalf("move product to event: %v", err)
	}
	return activityID, sessionID, productID
}

func cleanupOrder(t *testing.T, pool *pgxpool.Pool, orderID string) {
	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = pool.Exec(ctx, `DELETE FROM validation_log WHERE entitlement_id IN (SELECT id FROM entitlements WHERE order_id = $1)`, orderID)
		_, _ = pool.Exec(ctx, `DELETE FROM entitlement_items WHERE order_item_id IN (SELECT id FROM order_items WHERE order_id = $1)`, orderID)
		_, _ = pool.Exec(ctx, `DELETE FROM entitlements WHERE order_id = $1`, orderID)
		_, _ = pool.Exec(ctx, `DELETE FROM payments WHERE order_id = $1`, orderID)
		_, _ = pool.Exec(ctx, `DELETE FROM order_items WHERE order_id = $1`, orderID)
		_, _ = pool.Exec(ctx, `DELETE FROM orders WHERE id = $1`, orderID)
	})
}

func testProduct(productID, activityID string, priceCents int) product.Product {
	return product.Product{
		ID: productID, Type: product.TypeClass, PriceCents: priceCents,
		Activities: []product.Activity{{ID: activityID, Title: "Atividade Teste"}},
	}
}

func TestEvents_PublicCatalogIsScopedPerEvent(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	catalog := NewCatalogRepository(pool)
	sessions := NewSessionRepository(pool)

	eventB := mustInsertEvent(t, pool, true)
	activityB, _, productB := mustInsertEventCatalog(t, pool, eventB)

	mainProducts, err := catalog.ListPublicProducts(ctx, DefaultEventID)
	if err != nil {
		t.Fatalf("ListPublicProducts(main): %v", err)
	}
	for _, p := range mainProducts {
		if p.ID == productB {
			t.Fatal("event B's product leaked into the main event's catalog")
		}
	}

	bProducts, err := catalog.ListPublicProducts(ctx, eventB)
	if err != nil {
		t.Fatalf("ListPublicProducts(B): %v", err)
	}
	if len(bProducts) != 1 || bProducts[0].ID != productB || bProducts[0].EventID != eventB {
		t.Fatalf("event B catalog = %+v, want only its own product", bProducts)
	}

	mainNext, err := sessions.NextPerActivity(ctx, DefaultEventID)
	if err != nil {
		t.Fatalf("NextPerActivity(main): %v", err)
	}
	for _, s := range mainNext {
		if s.ActivityID == activityB {
			t.Fatal("event B's date leaked into the main event's page")
		}
	}
	bNext, err := sessions.NextPerActivity(ctx, eventB)
	if err != nil {
		t.Fatalf("NextPerActivity(B): %v", err)
	}
	if len(bNext) != 1 || bNext[0].ActivityID != activityB {
		t.Fatalf("event B next sessions = %+v, want only its own turma", bNext)
	}
}

func TestEvents_UnpublishedEventProductIsNotSold(t *testing.T) {
	pool := testPool(t)
	catalog := NewCatalogRepository(pool)

	draft := mustInsertEvent(t, pool, false)
	_, _, productID := mustInsertEventCatalog(t, pool, draft)

	if _, err := catalog.GetProductByID(context.Background(), productID); err == nil {
		t.Fatal("a product of an unpublished event must not be purchasable")
	}
}

func TestEvents_OrderIsRecordedUnderItsProductEvent(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	orders := NewOrderRepository(pool)

	eventB := mustInsertEvent(t, pool, true)
	activityB, sessionB, productB := mustInsertEventCatalog(t, pool, eventB)
	studentID := mustInsertStudent(t, pool)

	order, err := orders.CreateOrder(ctx, studentID, testProduct(productB, activityB, 30000), map[string]string{activityB: sessionB})
	if err != nil {
		t.Fatalf("CreateOrder: %v", err)
	}
	cleanupOrder(t, pool, order.ID)

	var eventID string
	if err := pool.QueryRow(ctx, `SELECT event_id FROM orders WHERE id = $1`, order.ID).Scan(&eventID); err != nil {
		t.Fatalf("read order: %v", err)
	}
	if eventID != eventB {
		t.Fatalf("order.event_id = %s, want %s (the product's event)", eventID, eventB)
	}

	// The main event's products keep landing in the main event, exactly as before.
	activityMain := mustInsertActivity(t, pool, vendorP5)
	sessionMain := mustInsertSession(t, pool, activityMain, 10)
	productMain := mustInsertProduct(t, pool, activityMain)
	mainOrder, err := orders.CreateOrder(ctx, studentID, testProduct(productMain, activityMain, 1000), map[string]string{activityMain: sessionMain})
	if err != nil {
		t.Fatalf("CreateOrder(main): %v", err)
	}
	cleanupOrder(t, pool, mainOrder.ID)
	if err := pool.QueryRow(ctx, `SELECT event_id FROM orders WHERE id = $1`, mainOrder.ID).Scan(&eventID); err != nil {
		t.Fatalf("read main order: %v", err)
	}
	if eventID != DefaultEventID {
		t.Fatalf("main order.event_id = %s, want the main event", eventID)
	}
}

func TestEvents_VoucherOnlyRedeemsItsOwnEvent(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	vouchers := NewVoucherRepository(pool)

	eventB := mustInsertEvent(t, pool, true)
	teamMemberID := mustInsertTeamMember(t, pool, "admin", vendorP5)
	studentID := mustInsertStudent(t, pool)

	// A voucher created in the main event…
	voucherID, code, err := vouchers.Create(ctx, DefaultEventID, "Cortesia", "Empresa", teamMemberID)
	if err != nil {
		t.Fatalf("Create voucher: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM vouchers WHERE id = $1`, voucherID) })

	// …can't pay for a product of event B, and isn't consumed by the attempt.
	if _, err := vouchers.Claim(ctx, code, studentID, eventB); !errors.Is(err, ErrVoucherOtherEvent) {
		t.Fatalf("Claim for another event's product: got %v, want ErrVoucherOtherEvent", err)
	}
	var status string
	if err := pool.QueryRow(ctx, `SELECT status FROM vouchers WHERE id = $1`, voucherID).Scan(&status); err != nil {
		t.Fatalf("read voucher: %v", err)
	}
	if status != "available" {
		t.Fatalf("voucher status after refused claim = %s, want available", status)
	}

	if _, err := vouchers.Claim(ctx, code, studentID, DefaultEventID); err != nil {
		t.Fatalf("Claim for its own event: %v", err)
	}

	// The Admin's voucher list of event B doesn't show the main event's voucher.
	listB, err := vouchers.List(ctx, eventB)
	if err != nil {
		t.Fatalf("List(B): %v", err)
	}
	for _, v := range listB {
		if v.ID == voucherID {
			t.Fatal("main event voucher leaked into event B's voucher list")
		}
	}
}

func TestEvents_CheckinRejectsTicketOfAnotherEvent(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	orders := NewOrderRepository(pool)
	checkin := NewCheckinRepository(pool)

	eventB := mustInsertEvent(t, pool, true)
	activityB, sessionB, productB := mustInsertEventCatalog(t, pool, eventB)
	studentID := mustInsertStudent(t, pool)
	staffID := mustInsertTeamMember(t, pool, "staff", vendorP5)

	order, err := orders.CreateOrder(ctx, studentID, testProduct(productB, activityB, 30000), map[string]string{activityB: sessionB})
	if err != nil {
		t.Fatalf("CreateOrder: %v", err)
	}
	cleanupOrder(t, pool, order.ID)
	paid, err := orders.MarkPaidAndIssueEntitlements(ctx, order.ID, "pay_"+uuid.NewString()[:8], "pix", 30000, "test-qr-secret")
	if err != nil {
		t.Fatalf("MarkPaidAndIssueEntitlements: %v", err)
	}
	if paid.EventName != "Evento Teste" || paid.EventSlug == "" {
		t.Errorf("paid order event label = %q/%q, want the event's name/slug for the e-mail", paid.EventName, paid.EventSlug)
	}
	token := paid.Entitlements[0].QRToken
	entitlementID := paid.Entitlements[0].ID

	var validUntil time.Time
	if err := pool.QueryRow(ctx, `SELECT valid_until FROM entitlements WHERE id = $1`, entitlementID).Scan(&validUntil); err != nil {
		t.Fatalf("read entitlement: %v", err)
	}
	day := validUntil.Format("2006-01-02")

	// The main event's roster for that day doesn't list event B's guest…
	mainRoster, err := checkin.ListRoster(ctx, "", day, DefaultEventID)
	if err != nil {
		t.Fatalf("ListRoster(main): %v", err)
	}
	for _, g := range mainRoster {
		for _, e := range g.Entries {
			if e.EntitlementID == entitlementID {
				t.Fatal("event B's guest leaked into the main event's check-in list")
			}
		}
	}
	// …event B's does.
	bRoster, err := checkin.ListRoster(ctx, "", day, eventB)
	if err != nil {
		t.Fatalf("ListRoster(B): %v", err)
	}
	found := false
	for _, g := range bRoster {
		for _, e := range g.Entries {
			found = found || e.EntitlementID == entitlementID
		}
	}
	if !found {
		t.Fatal("event B's guest missing from event B's check-in list")
	}

	// Scanned on a terminal set to the main event: refused, and left unused.
	ticket, err := checkin.ValidateTokenForEvent(ctx, token, DefaultEventID, "", staffID, uuid.NewString(), validUntil)
	if err != nil {
		t.Fatalf("ValidateTokenForEvent(main): %v", err)
	}
	if ticket.Result != ResultWrongEvent || ticket.EventName != "Evento Teste" {
		t.Fatalf("result = %s (event %q), want wrong_event naming the ticket's event", ticket.Result, ticket.EventName)
	}
	var status string
	if err := pool.QueryRow(ctx, `SELECT status FROM entitlements WHERE id = $1`, entitlementID).Scan(&status); err != nil {
		t.Fatalf("read entitlement status: %v", err)
	}
	if status != "available" {
		t.Fatalf("ticket status after wrong-event scan = %s, want available", status)
	}

	// Scanned on its own event's terminal: accepted.
	ticket, err = checkin.ValidateTokenForEvent(ctx, token, eventB, "", staffID, uuid.NewString(), validUntil)
	if err != nil {
		t.Fatalf("ValidateTokenForEvent(B): %v", err)
	}
	if ticket.Result != ResultSuccess {
		t.Fatalf("result on its own event = %s, want success", ticket.Result)
	}

	// Event B's dashboard counts its own sale.
	summary, err := NewDashboardRepository(pool).Summary(ctx, eventB, nil, nil)
	if err != nil {
		t.Fatalf("Summary(B): %v", err)
	}
	if summary.PaidOrders != 1 || summary.TotalRevenueCents != 30000 {
		t.Fatalf("event B summary = %+v, want 1 paid order / R$ 300,00", summary)
	}
}

func TestEvents_ProductCannotUseActivityOfAnotherEvent(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	admin := NewAdminCatalogRepository(pool)

	eventB := mustInsertEvent(t, pool, true)
	mainActivity := mustInsertActivity(t, pool, vendorP5)

	_, err := admin.CreateProduct(ctx, CreateProductParams{
		EventID: eventB, Title: "Ingresso B", Type: "class", PriceCents: 1000, ActivityIDs: []string{mainActivity},
	})
	if !errors.Is(err, ErrOtherEventActivity) {
		t.Fatalf("CreateProduct with another event's activity: got %v, want ErrOtherEventActivity", err)
	}

	// Editing a main-event product to use an activity of event B is refused too.
	activityB, _, _ := mustInsertEventCatalog(t, pool, eventB)
	mainProduct := mustInsertProduct(t, pool, mainActivity)
	err = admin.UpdateProduct(ctx, mainProduct, CreateProductParams{
		Title: "Produto Teste", Type: "class", PriceCents: 1000, ActivityIDs: []string{activityB},
	})
	if !errors.Is(err, ErrOtherEventActivity) {
		t.Fatalf("UpdateProduct with another event's activity: got %v, want ErrOtherEventActivity", err)
	}
}
