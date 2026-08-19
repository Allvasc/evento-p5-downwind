package handlers

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"p5wellness/backend/internal/asaas"
	"p5wellness/backend/internal/repository/postgres"
)

// DevToolsHandler is only mounted when APP_ENV != production. It lets the checkout →
// webhook → entitlements flow be exercised end to end without real Asaas sandbox
// credentials, by synthesizing the same confirmation event the real webhook would send.
type DevToolsHandler struct {
	orders  *postgres.OrderRepository
	webhook *WebhookAsaasHandler
	fake    *asaas.FakeClient
	log     *slog.Logger
}

func NewDevToolsHandler(orders *postgres.OrderRepository, webhook *WebhookAsaasHandler, fake *asaas.FakeClient, log *slog.Logger) *DevToolsHandler {
	return &DevToolsHandler{orders: orders, webhook: webhook, fake: fake, log: log}
}

func (h *DevToolsHandler) SimulatePaymentConfirmation(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "id")

	asaasPaymentID, _, err := h.orders.FindByID(r.Context(), orderID)
	if err != nil || asaasPaymentID == "" {
		writeJSONError(w, http.StatusNotFound, "pedido não encontrado ou sem cobrança associada")
		return
	}

	if h.fake != nil {
		h.fake.SimulateConfirmation(asaasPaymentID)
	}

	event := asaas.WebhookEvent{Event: "PAYMENT_CONFIRMED"}
	event.Payment.ID = asaasPaymentID
	event.Payment.Status = "CONFIRMED"

	h.webhook.process(r.Context(), event)
	writeJSON(w, http.StatusOK, map[string]string{"status": "simulated"})
}
