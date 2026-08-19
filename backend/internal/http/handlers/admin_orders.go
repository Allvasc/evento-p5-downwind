package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"p5wellness/backend/internal/asaas"
	"p5wellness/backend/internal/http/middleware"
	"p5wellness/backend/internal/mailer"
	"p5wellness/backend/internal/repository/postgres"
)

type AdminOrdersHandler struct {
	repo   *postgres.AdminOrderRepository
	orders *postgres.OrderRepository
	asaas  asaas.Client
	mailer mailer.Sender
	log    *slog.Logger
}

func NewAdminOrdersHandler(repo *postgres.AdminOrderRepository, orders *postgres.OrderRepository, asaasClient asaas.Client, sender mailer.Sender, log *slog.Logger) *AdminOrdersHandler {
	return &AdminOrdersHandler{repo: repo, orders: orders, asaas: asaasClient, mailer: sender, log: log}
}

func (h *AdminOrdersHandler) List(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	search := r.URL.Query().Get("q")
	from, to := parseDateRange(r)
	orders, err := h.repo.List(r.Context(), status, search, from, to)
	if err != nil {
		h.log.Error("list orders", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "não foi possível carregar os pedidos")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"orders": orders})
}

func (h *AdminOrdersHandler) Detail(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	detail, err := h.repo.Detail(r.Context(), id)
	if err != nil {
		if err == postgres.ErrNotFound {
			writeJSONError(w, http.StatusNotFound, "pedido não encontrado")
			return
		}
		h.log.Error("order detail", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "não foi possível carregar o pedido")
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

// Refund reverses the Asaas payment (when one exists — a pending order that never got
// a payment ID has nothing to reverse there) and, only once that succeeds, flips the
// order to refunded and cancels every not-yet-used entitlement.
func (h *AdminOrdersHandler) Refund(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	asaasPaymentID, status, err := h.orders.FindByID(r.Context(), id)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "pedido não encontrado")
		return
	}
	if status == "refunded" {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	}
	if status != "paid" {
		writeJSONError(w, http.StatusConflict, "só é possível estornar pedidos pagos")
		return
	}

	if asaasPaymentID != "" {
		if err := h.asaas.RefundPayment(r.Context(), asaasPaymentID); err != nil {
			h.log.Error("asaas refund payment", "error", err, "orderId", id)
			writeJSONError(w, http.StatusBadGateway, "não foi possível estornar o pagamento na Asaas")
			return
		}
	}

	if err := h.orders.RefundOrder(r.Context(), id); err != nil {
		h.log.Error("refund order", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "pagamento estornado na Asaas, mas houve um erro ao atualizar o pedido — verifique manualmente")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type rescheduleRequest struct {
	NewDate string `json:"newDate"`
	Reason  string `json:"reason"`
}

// Reschedule moves every benefit of a paid order to a new date — the "cliente não
// compareceu, quer remarcar" flow. Who did it and why is captured in order_reschedules
// (backend/internal/repository/postgres/order.go) and surfaced back in the order detail;
// the customer is best-effort notified by e-mail with the new date once the change lands.
func (h *AdminOrdersHandler) Reschedule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	claims := middleware.ClaimsFromContext(r.Context())

	var req rescheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.NewDate == "" || strings.TrimSpace(req.Reason) == "" {
		writeJSONError(w, http.StatusBadRequest, "informe a nova data e o motivo da remarcação")
		return
	}

	result, err := h.orders.Reschedule(r.Context(), id, req.NewDate, strings.TrimSpace(req.Reason), claims.UserID())
	if err != nil {
		switch {
		case errors.Is(err, postgres.ErrNotFound):
			writeJSONError(w, http.StatusNotFound, "pedido não encontrado")
		case errors.Is(err, postgres.ErrOrderNotPaid):
			writeJSONError(w, http.StatusConflict, "só é possível remarcar pedidos pagos")
		case errors.Is(err, postgres.ErrTicketAlreadyUsed):
			writeJSONError(w, http.StatusConflict, "um dos benefícios deste pedido já foi utilizado, não é possível remarcar")
		case errors.Is(err, postgres.ErrRescheduleDateInPast):
			writeJSONError(w, http.StatusBadRequest, "a nova data não pode ser no passado")
		case errors.Is(err, postgres.ErrNoSessionOnDate):
			writeJSONError(w, http.StatusConflict, "não há turma disponível nessa data para uma das atividades do pedido")
		case errors.Is(err, postgres.ErrSessionFull):
			writeJSONError(w, http.StatusConflict, "a turma dessa data já está lotada")
		default:
			h.log.Error("reschedule order", "error", err, "orderId", id)
			writeJSONError(w, http.StatusInternalServerError, "não foi possível remarcar o pedido")
		}
		return
	}

	if detail, err := h.orders.GetPaidOrderDetail(r.Context(), id); err != nil {
		h.log.Error("get order detail for reschedule email", "error", err, "orderId", id)
	} else if err := sendRescheduleEmail(r.Context(), h.mailer, h.log, *detail, result.NewDate); err != nil {
		h.log.Error("send reschedule email", "error", err, "orderId", id)
	}

	writeJSON(w, http.StatusOK, map[string]string{"previousDate": result.PreviousDate, "newDate": result.NewDate})
}

// ResendEmail is the admin-triggered counterpart of the student's self-service resend —
// useful when support fields a "não recebi o QR" complaint directly.
func (h *AdminOrdersHandler) ResendEmail(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	detail, err := h.orders.GetPaidOrderDetail(r.Context(), id)
	if err != nil {
		if err == postgres.ErrNotFound {
			writeJSONError(w, http.StatusNotFound, "pedido não encontrado")
			return
		}
		if err == postgres.ErrOrderNotPaid {
			writeJSONError(w, http.StatusConflict, "esse pedido ainda não foi pago")
			return
		}
		h.log.Error("get paid order detail", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "não foi possível reenviar o e-mail")
		return
	}

	if err := sendTicketEmail(r.Context(), h.mailer, h.log, *detail); err != nil {
		h.log.Error("resend ticket email", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "não foi possível reenviar o e-mail")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
