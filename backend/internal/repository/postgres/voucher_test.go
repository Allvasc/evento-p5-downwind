package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"p5wellness/backend/internal/domain/product"
)

func mustCreateVoucher(t *testing.T, pool *pgxpool.Pool, repo *VoucherRepository, createdBy string) (id, code string) {
	t.Helper()
	id, code, err := repo.Create(context.Background(), "Campanha Teste", "Empresa Teste", createdBy)
	if err != nil {
		t.Fatalf("create voucher: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM vouchers WHERE id = $1`, id) })
	return id, code
}

func TestVoucher_ClaimReleaseRoundTrip(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	repo := NewVoucherRepository(pool)

	teamMemberID := mustInsertTeamMember(t, pool, "admin", vendorAYO)
	studentID := mustInsertStudent(t, pool)
	_, code := mustCreateVoucher(t, pool, repo, teamMemberID)

	voucherID, err := repo.Claim(ctx, code, studentID)
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}

	var status, redeemedBy string
	if err := pool.QueryRow(ctx, `SELECT status, redeemed_by FROM vouchers WHERE id = $1`, voucherID).Scan(&status, &redeemedBy); err != nil {
		t.Fatalf("read voucher: %v", err)
	}
	if status != "used" || redeemedBy != studentID {
		t.Fatalf("after Claim: status=%s redeemedBy=%s, want used/%s", status, redeemedBy, studentID)
	}

	// A second claim on the same code must fail — vouchers are single-use.
	otherStudent := mustInsertStudent(t, pool)
	if _, err := repo.Claim(ctx, code, otherStudent); !errors.Is(err, ErrVoucherNotAvailable) {
		t.Fatalf("second Claim: got %v, want ErrVoucherNotAvailable", err)
	}

	// Release puts it back exactly as it was — e.g. after a downstream booking failure —
	// so the code is reusable rather than wasted.
	if err := repo.Release(ctx, voucherID); err != nil {
		t.Fatalf("Release: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT status FROM vouchers WHERE id = $1`, voucherID).Scan(&status); err != nil {
		t.Fatalf("read voucher after release: %v", err)
	}
	if status != "available" {
		t.Fatalf("after Release: status=%s, want available", status)
	}

	if _, err := repo.Claim(ctx, code, otherStudent); err != nil {
		t.Fatalf("Claim after Release: %v", err)
	}
}

func TestVoucher_ClaimUnknownCode(t *testing.T) {
	pool := testPool(t)
	repo := NewVoucherRepository(pool)
	studentID := mustInsertStudent(t, pool)

	if _, err := repo.Claim(context.Background(), "P5-DOESNOTEXIST", studentID); !errors.Is(err, ErrVoucherNotFound) {
		t.Fatalf("got %v, want ErrVoucherNotFound", err)
	}
}

func TestVoucher_CancelRules(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	repo := NewVoucherRepository(pool)
	teamMemberID := mustInsertTeamMember(t, pool, "admin", vendorAYO)

	// Cancelling an untouched voucher works, and cancelling it again is a harmless no-op.
	id1, _ := mustCreateVoucher(t, pool, repo, teamMemberID)
	if err := repo.Cancel(ctx, id1); err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	if err := repo.Cancel(ctx, id1); err != nil {
		t.Fatalf("Cancel again (idempotent): %v", err)
	}

	// A redeemed voucher can't be cancelled out from under the customer who already used it.
	id2, code2 := mustCreateVoucher(t, pool, repo, teamMemberID)
	studentID := mustInsertStudent(t, pool)
	if _, err := repo.Claim(ctx, code2, studentID); err != nil {
		t.Fatalf("Claim: %v", err)
	}
	if err := repo.Cancel(ctx, id2); !errors.Is(err, ErrVoucherHasRedemption) {
		t.Fatalf("Cancel redeemed voucher: got %v, want ErrVoucherHasRedemption", err)
	}
}

// TestVoucher_RedemptionIssuesFreeEntitlement exercises the same repository calls the
// HTTP handler chains together (voucher_student.go): claim, book a real order+session via
// the normal CreateOrder path, mark it paid with billing_type "voucher" and amountCents 0,
// then link the voucher to the resulting order — confirming a voucher redemption ends up
// indistinguishable from a paid order except for how it was paid for.
func TestVoucher_RedemptionIssuesFreeEntitlement(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	voucherRepo := NewVoucherRepository(pool)
	orderRepo := NewOrderRepository(pool)

	teamMemberID := mustInsertTeamMember(t, pool, "admin", vendorAYO)
	studentID := mustInsertStudent(t, pool)
	activityID := mustInsertActivity(t, pool, vendorAYO)
	sessionID := mustInsertSession(t, pool, activityID, 10)
	productID := mustInsertProduct(t, pool, activityID)

	_, code := mustCreateVoucher(t, pool, voucherRepo, teamMemberID)

	voucherID, err := voucherRepo.Claim(ctx, code, studentID)
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}

	p := product.Product{
		ID:         productID,
		Type:       product.TypeClass,
		PriceCents: 2500,
		Activities: []product.Activity{{ID: activityID, Title: "Atividade Teste"}},
	}
	order, err := orderRepo.CreateOrder(ctx, studentID, p, map[string]string{activityID: sessionID})
	if err != nil {
		t.Fatalf("CreateOrder: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM entitlement_items WHERE order_item_id IN (SELECT id FROM order_items WHERE order_id = $1)`, order.ID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM entitlements WHERE order_id = $1`, order.ID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM payments WHERE order_id = $1`, order.ID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM order_items WHERE order_id = $1`, order.ID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM orders WHERE id = $1`, order.ID)
	})

	if err := orderRepo.SetAsaasPayment(ctx, order.ID, "voucher", code); err != nil {
		t.Fatalf("SetAsaasPayment: %v", err)
	}

	result, err := orderRepo.MarkPaidAndIssueEntitlements(ctx, order.ID, code, "voucher", 0, "test-qr-secret")
	if err != nil {
		t.Fatalf("MarkPaidAndIssueEntitlements: %v", err)
	}
	if len(result.Entitlements) != 1 {
		t.Fatalf("got %d entitlements, want 1", len(result.Entitlements))
	}

	if err := voucherRepo.LinkOrder(ctx, voucherID, order.ID); err != nil {
		t.Fatalf("LinkOrder: %v", err)
	}

	var billingType string
	var amountCents int
	if err := pool.QueryRow(ctx, `SELECT billing_type, amount_cents FROM payments WHERE order_id = $1`, order.ID).Scan(&billingType, &amountCents); err != nil {
		t.Fatalf("read payment: %v", err)
	}
	if billingType != "voucher" || amountCents != 0 {
		t.Errorf("payment billing_type/amount = %s/%d, want voucher/0", billingType, amountCents)
	}

	var paymentMethod string
	if err := pool.QueryRow(ctx, `SELECT payment_method FROM orders WHERE id = $1`, order.ID).Scan(&paymentMethod); err != nil {
		t.Fatalf("read order: %v", err)
	}
	if paymentMethod != "voucher" {
		t.Errorf("order.payment_method = %s, want voucher", paymentMethod)
	}

	var linkedOrderID string
	if err := pool.QueryRow(ctx, `SELECT order_id FROM vouchers WHERE id = $1`, voucherID).Scan(&linkedOrderID); err != nil {
		t.Fatalf("read voucher: %v", err)
	}
	if linkedOrderID != order.ID {
		t.Errorf("voucher.order_id = %s, want %s", linkedOrderID, order.ID)
	}
}
