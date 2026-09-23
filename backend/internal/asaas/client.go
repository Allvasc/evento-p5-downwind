package asaas

import (
	"context"
	"errors"
)

// ErrNotFound is returned when Asaas answers 404 for a payment — e.g. a charge that was
// already deleted, so it can no longer be paid.
var ErrNotFound = errors.New("asaas: not found")

// Client abstracts the Asaas payment API so the checkout/webhook handlers never depend
// on whether they're talking to the real sandbox/production API or a local fake used
// while no API key is configured.
type Client interface {
	CreateCustomer(ctx context.Context, p CustomerParams) (customerID string, err error)
	CreateCharge(ctx context.Context, p ChargeParams) (Charge, error)
	GetPayment(ctx context.Context, paymentID string) (Payment, error)
	RefundPayment(ctx context.Context, paymentID string) error
	// DeletePayment cancels a pending charge so its Pix QR code can no longer be paid.
	// Asaas refuses it for a charge that was already received.
	DeletePayment(ctx context.Context, paymentID string) error
}
