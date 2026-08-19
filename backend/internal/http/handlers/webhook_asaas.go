package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"p5wellness/backend/internal/asaas"
	"p5wellness/backend/internal/mailer"
	"p5wellness/backend/internal/repository/postgres"
)

type WebhookAsaasHandler struct {
	orders       *postgres.OrderRepository
	webhooks     *postgres.WebhookRepository
	mailer       mailer.Sender
	webhookToken string
	qrSecret     string
	log          *slog.Logger
}

func NewWebhookAsaasHandler(orders *postgres.OrderRepository, webhooks *postgres.WebhookRepository, sender mailer.Sender, webhookToken, qrSecret string, log *slog.Logger) *WebhookAsaasHandler {
	return &WebhookAsaasHandler{orders: orders, webhooks: webhooks, mailer: sender, webhookToken: webhookToken, qrSecret: qrSecret, log: log}
}

var confirmedEvents = map[string]bool{
	"PAYMENT_CONFIRMED": true,
	"PAYMENT_RECEIVED":  true,
}

var failedEvents = map[string]bool{
	"PAYMENT_OVERDUE":  true,
	"PAYMENT_DELETED":  true,
	"PAYMENT_REFUNDED": true,
}

func (h *WebhookAsaasHandler) Handle(w http.ResponseWriter, r *http.Request) {
	if h.webhookToken != "" && r.Header.Get("asaas-access-token") != h.webhookToken {
		writeJSONError(w, http.StatusUnauthorized, "token inválido")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "corpo inválido")
		return
	}

	sum := sha256.Sum256(body)
	dedupeHash := hex.EncodeToString(sum[:])

	var event asaas.WebhookEvent
	if err := json.Unmarshal(body, &event); err != nil {
		writeJSONError(w, http.StatusBadRequest, "payload inválido")
		return
	}

	isNew, err := h.webhooks.RecordIfNew(r.Context(), "asaas", dedupeHash, event.Event, body)
	if err != nil {
		h.log.Error("record webhook", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "erro ao processar")
		return
	}
	if !isNew {
		writeJSON(w, http.StatusOK, map[string]string{"status": "duplicate, ignored"})
		return
	}

	h.process(r.Context(), event)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *WebhookAsaasHandler) process(ctx context.Context, event asaas.WebhookEvent) {
	paymentID := event.Payment.ID
	if paymentID == "" {
		return
	}

	switch {
	case confirmedEvents[event.Event]:
		orderID, err := h.orders.FindOrderIDByAsaasPaymentID(ctx, paymentID)
		if err != nil {
			h.log.Warn("webhook: order not found for payment", "paymentId", paymentID)
			return
		}
		result, err := h.orders.MarkPaidAndIssueEntitlements(ctx, orderID, paymentID, "pix", int(event.Payment.Value*100), h.qrSecret)
		if err != nil {
			h.log.Error("mark paid", "error", err, "orderId", orderID)
			return
		}
		if result.AlreadyPaid {
			return
		}
		h.sendConfirmationEmail(ctx, *result)

	case failedEvents[event.Event]:
		if err := h.orders.MarkStatusByAsaasPaymentID(ctx, paymentID, statusForFailedEvent(event.Event)); err != nil {
			h.log.Error("mark failed", "error", err)
		}

	default:
		h.log.Info("webhook: unhandled event", "event", event.Event)
	}
}

func statusForFailedEvent(event string) string {
	if event == "PAYMENT_REFUNDED" {
		return "refunded"
	}
	return "failed"
}

func (h *WebhookAsaasHandler) sendConfirmationEmail(ctx context.Context, result postgres.PaidOrderResult) {
	if err := sendTicketEmail(ctx, h.mailer, h.log, result); err != nil {
		h.log.Error("send confirmation email", "error", err)
	}
}
