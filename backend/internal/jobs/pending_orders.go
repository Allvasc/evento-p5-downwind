package jobs

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"p5wellness/backend/internal/asaas"
	"p5wellness/backend/internal/repository/postgres"
)

// paidStatuses are Asaas payment statuses meaning the customer already paid — the
// PAYMENT_CONFIRMED/RECEIVED webhook will mark the order paid, so it must not expire.
var paidStatuses = map[string]bool{
	"CONFIRMED":              true,
	"RECEIVED":               true,
	"RECEIVED_IN_CASH":       true,
	"DUNNING_RECEIVED":       true,
	"AWAITING_RISK_ANALYSIS": true,
}

// PendingOrderExpirer releases the seats of Pix orders left unpaid for longer than
// postgres.PendingOrderTTL: it deletes the Asaas charge first (so the QR code can no
// longer be paid) and only then marks the order expired. If Asaas refuses the deletion
// — e.g. the customer paid in the last second — the order is left pending for the
// webhook to confirm, so a paid order never loses its seat.
type PendingOrderExpirer struct {
	orders   *postgres.OrderRepository
	asaas    asaas.Client
	log      *slog.Logger
	ttl      time.Duration
	interval time.Duration
}

func NewPendingOrderExpirer(orders *postgres.OrderRepository, asaasClient asaas.Client, log *slog.Logger) *PendingOrderExpirer {
	return &PendingOrderExpirer{orders: orders, asaas: asaasClient, log: log, ttl: postgres.PendingOrderTTL, interval: 30 * time.Second}
}

// Run sweeps every interval until ctx is cancelled.
func (e *PendingOrderExpirer) Run(ctx context.Context) {
	ticker := time.NewTicker(e.interval)
	defer ticker.Stop()
	for {
		e.Sweep(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (e *PendingOrderExpirer) Sweep(ctx context.Context) {
	stale, err := e.orders.ListStalePending(ctx, e.ttl)
	if err != nil {
		e.log.Error("expirer: list stale pending orders", "error", err)
		return
	}
	for _, o := range stale {
		if ctx.Err() != nil {
			return
		}
		e.expire(ctx, o)
	}
}

func (e *PendingOrderExpirer) expire(ctx context.Context, o postgres.StalePendingOrder) {
	if o.AsaasPaymentID != "" {
		payment, err := e.asaas.GetPayment(ctx, o.AsaasPaymentID)
		switch {
		case errors.Is(err, asaas.ErrNotFound):
			// Charge no longer exists in Asaas, so it can't be paid — just release the seat.
		case err != nil:
			e.log.Warn("expirer: get payment failed, retrying next sweep", "error", err, "order", o.OrderNumber)
			return
		case paidStatuses[payment.Status]:
			return
		case payment.Deleted:
			// Already removed (e.g. a previous sweep deleted it but failed to mark expired).
		case payment.Status == "PENDING" || payment.Status == "OVERDUE":
			if err := e.asaas.DeletePayment(ctx, o.AsaasPaymentID); err != nil && !errors.Is(err, asaas.ErrNotFound) {
				e.log.Warn("expirer: delete payment failed, keeping order pending", "error", err, "order", o.OrderNumber)
				return
			}
		}
	}

	if err := e.orders.MarkExpired(ctx, o.ID); err != nil {
		e.log.Error("expirer: mark expired", "error", err, "order", o.OrderNumber)
		return
	}
	e.log.Info("expirer: pix não pago em 10 min, pedido expirado e vaga liberada", "order", o.OrderNumber)
}
