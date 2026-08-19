package asaas

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
)

// FakeClient stands in for the real Asaas API when no ASAAS_API_KEY is configured, so
// the checkout → webhook → entitlements flow can be built and tested end to end without
// sandbox credentials. Swap in HTTPClient once real keys are available — same interface.
type FakeClient struct {
	mu       sync.Mutex
	payments map[string]string // paymentID -> status
}

func NewFakeClient() *FakeClient {
	return &FakeClient{payments: map[string]string{}}
}

func (c *FakeClient) CreateCustomer(_ context.Context, p CustomerParams) (string, error) {
	return "fake_cus_" + uuid.NewString()[:8], nil
}

func (c *FakeClient) CreateCharge(_ context.Context, p ChargeParams) (Charge, error) {
	id := "fake_pay_" + uuid.NewString()[:12]
	c.mu.Lock()
	c.payments[id] = "PENDING"
	c.mu.Unlock()

	return Charge{
		PaymentID:    id,
		Status:       "PENDING",
		PixQRImage:   "",
		PixCopyPaste: fmt.Sprintf("00020126580014BR.GOV.BCB.PIX-FAKE-%s-VALOR-%d", id, p.ValueCents),
	}, nil
}

func (c *FakeClient) GetPayment(_ context.Context, paymentID string) (Payment, error) {
	c.mu.Lock()
	status := c.payments[paymentID]
	c.mu.Unlock()
	return Payment{ID: paymentID, Status: status}, nil
}

func (c *FakeClient) RefundPayment(_ context.Context, paymentID string) error {
	c.mu.Lock()
	c.payments[paymentID] = "REFUNDED"
	c.mu.Unlock()
	return nil
}

// SimulateConfirmation is a dev-only helper (wired to a /dev/* route, never mounted in
// production) that flips a fake charge to CONFIRMED so the webhook path can be exercised.
func (c *FakeClient) SimulateConfirmation(paymentID string) {
	c.mu.Lock()
	c.payments[paymentID] = "CONFIRMED"
	c.mu.Unlock()
}
