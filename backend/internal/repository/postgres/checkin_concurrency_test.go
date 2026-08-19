package postgres

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"p5wellness/backend/internal/qrcode"
)

// TestValidateToken_ConcurrentScansExactlyOneSucceeds is the test the original plan
// called for and the codebase never had: two staff devices scanning the same QR ticket
// at (as close to) the same instant must not both succeed. ValidateToken relies on
// `UPDATE ... WHERE status = 'available'` for this — a plain SELECT-then-UPDATE would
// let both goroutines read "available" before either writes, double-validating the
// ticket. This test fails loudly if that atomicity ever regresses.
func TestValidateToken_ConcurrentScansExactlyOneSucceeds(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	repo := NewCheckinRepository(pool)

	studentID := mustInsertStudent(t, pool)
	activityID := mustInsertActivity(t, pool, vendorAYO)
	sessionID := mustInsertSession(t, pool, activityID, 10)
	orderID := mustInsertPaidOrder(t, pool, studentID)
	orderItemID := mustInsertOrderItem(t, pool, orderID, activityID, sessionID)
	teamMemberID := mustInsertTeamMember(t, pool, "staff", vendorAYO)

	token, _, err := qrcode.GenerateToken("test-secret")
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	mustInsertEntitlement(t, pool, orderID, studentID, vendorAYO, token, orderItemID)

	const attempts = 5
	results := make([]ValidationResult, attempts)
	var wg sync.WaitGroup
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			ticket, err := repo.ValidateToken(ctx, token, vendorAYO, teamMemberID, uuid.NewString(), time.Now())
			if err != nil {
				t.Errorf("goroutine %d: ValidateToken error: %v", i, err)
				return
			}
			results[i] = ticket.Result
		}(i)
	}
	wg.Wait()

	successes, alreadyUsed := 0, 0
	for _, r := range results {
		switch r {
		case ResultSuccess:
			successes++
		case ResultAlreadyUsed:
			alreadyUsed++
		default:
			t.Errorf("unexpected result: %s", r)
		}
	}
	if successes != 1 {
		t.Errorf("got %d successful validations racing on the same ticket, want exactly 1 (got %d already_used)", successes, alreadyUsed)
	}
	if alreadyUsed != attempts-1 {
		t.Errorf("got %d already_used, want %d", alreadyUsed, attempts-1)
	}
}

// TestValidateToken_WrongSectorRejected locks in the sector-scoping rule: a staff
// member from one vendor (P5) must never be able to validate — or consume — a ticket
// that belongs to another vendor (AYO). The ticket must stay available afterwards for
// the correct sector to use.
func TestValidateToken_WrongSectorRejected(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	repo := NewCheckinRepository(pool)

	studentID := mustInsertStudent(t, pool)
	activityID := mustInsertActivity(t, pool, vendorAYO)
	sessionID := mustInsertSession(t, pool, activityID, 10)
	orderID := mustInsertPaidOrder(t, pool, studentID)
	orderItemID := mustInsertOrderItem(t, pool, orderID, activityID, sessionID)
	p5Staff := mustInsertTeamMember(t, pool, "staff", vendorP5)
	ayoStaff := mustInsertTeamMember(t, pool, "staff", vendorAYO)

	token, _, err := qrcode.GenerateToken("test-secret")
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	mustInsertEntitlement(t, pool, orderID, studentID, vendorAYO, token, orderItemID)

	ticket, err := repo.ValidateToken(ctx, token, vendorP5, p5Staff, uuid.NewString(), time.Now())
	if err != nil {
		t.Fatalf("ValidateToken (wrong sector): %v", err)
	}
	if ticket.Result != ResultWrongSector {
		t.Fatalf("got result %q, want %q", ticket.Result, ResultWrongSector)
	}

	ticket, err = repo.ValidateToken(ctx, token, vendorAYO, ayoStaff, uuid.NewString(), time.Now())
	if err != nil {
		t.Fatalf("ValidateToken (correct sector): %v", err)
	}
	if ticket.Result != ResultSuccess {
		t.Fatalf("got result %q after wrong-sector rejection, want %q (ticket should still be available)", ticket.Result, ResultSuccess)
	}
}
