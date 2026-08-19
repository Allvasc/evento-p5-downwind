package asaas

import "context"

// Client abstracts the Asaas payment API so the checkout/webhook handlers never depend
// on whether they're talking to the real sandbox/production API or a local fake used
// while no API key is configured.
type Client interface {
	CreateCustomer(ctx context.Context, p CustomerParams) (customerID string, err error)
	CreateCharge(ctx context.Context, p ChargeParams) (Charge, error)
	GetPayment(ctx context.Context, paymentID string) (Payment, error)
	RefundPayment(ctx context.Context, paymentID string) error
}
