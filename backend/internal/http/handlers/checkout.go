package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"p5wellness/backend/internal/asaas"
	"p5wellness/backend/internal/domain/product"
	"p5wellness/backend/internal/http/middleware"
	"p5wellness/backend/internal/mailer"
	"p5wellness/backend/internal/repository/postgres"
)

type CheckoutHandler struct {
	catalog       *postgres.CatalogRepository
	orders        *postgres.OrderRepository
	students      *postgres.StudentRepository
	asaas         asaas.Client
	mailer        mailer.Sender
	encryptionKey string
	qrSecret      string
	log           *slog.Logger
}

func NewCheckoutHandler(catalog *postgres.CatalogRepository, orders *postgres.OrderRepository, students *postgres.StudentRepository, asaasClient asaas.Client, sender mailer.Sender, encryptionKey, qrSecret string, log *slog.Logger) *CheckoutHandler {
	return &CheckoutHandler{catalog: catalog, orders: orders, students: students, asaas: asaasClient, mailer: sender, encryptionKey: encryptionKey, qrSecret: qrSecret, log: log}
}

type checkoutRequest struct {
	ProductID  string            `json:"productId"`
	SessionIDs map[string]string `json:"sessionIds"` // activityId -> class_session_id
}

type checkoutResponse struct {
	OrderID      string `json:"orderId"`
	OrderNumber  string `json:"orderNumber"`
	TotalCents   int    `json:"totalCents"`
	PixCopyPaste string `json:"pixCopyPaste"`
	PixQRImage   string `json:"pixQrImage,omitempty"`
	// Free: produto com preço R$ 0 — o pedido já sai pago, sem Pix (ver createFree).
	Free bool `json:"free,omitempty"`
}

func (h *CheckoutHandler) Create(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())

	var req checkoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ProductID == "" {
		writeJSONError(w, http.StatusBadRequest, "produto inválido")
		return
	}

	prod, err := h.catalog.GetProductByID(r.Context(), req.ProductID)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "experiência não encontrada")
		return
	}

	student, err := h.students.FindByID(r.Context(), claims.UserID())
	if err != nil {
		writeJSONError(w, http.StatusUnauthorized, "sessão inválida")
		return
	}

	// Contas criadas antes do contato de emergência ser obrigatório preenchem na compra.
	if !student.HasEmergencyContact() {
		writeJSONError(w, http.StatusBadRequest, errEmergencyContactRequired)
		return
	}

	if prod.PriceCents == 0 {
		h.createFree(w, r, student.ID, prod, req.SessionIDs)
		return
	}

	// A Asaas exige CPF/CNPJ no cliente para gerar qualquer cobrança — checar isso antes
	// de criar o pedido evita um pedido "pending" órfão quando a cobrança for recusada.
	cpf, err := h.students.DecryptedCPF(r.Context(), student.ID, h.encryptionKey)
	if err != nil {
		h.log.Error("decrypt cpf", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "não foi possível iniciar a compra")
		return
	}
	if cpf == "" {
		writeJSONError(w, http.StatusBadRequest, "para pagar com Pix/cartão precisamos do seu CPF — atualize seu cadastro antes de continuar")
		return
	}

	order, err := h.orders.CreateOrder(r.Context(), student.ID, *prod, req.SessionIDs)
	if err != nil {
		switch {
		case errors.Is(err, postgres.ErrActivityChoiceRequired):
			writeJSONError(w, http.StatusConflict, "escolha uma atividade antes de continuar")
			return
		case errors.Is(err, postgres.ErrSessionRequired):
			writeJSONError(w, http.StatusConflict, "escolha uma turma para: "+activityFromError(err))
			return
		case errors.Is(err, postgres.ErrSessionFull):
			writeJSONError(w, http.StatusConflict, "essa turma acabou de lotar, escolha outra data para: "+activityFromError(err))
			return
		}
		h.log.Error("create order", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "não foi possível iniciar a compra")
		return
	}

	customerID, err := h.asaas.CreateCustomer(r.Context(), asaas.CustomerParams{
		Name: student.FullName, Email: student.Email, Phone: student.Phone, CPF: cpf,
	})
	if err != nil {
		h.log.Error("asaas create customer", "error", err)
		// The order (and its seat reservation) was already committed by CreateOrder —
		// without this, a failed Asaas call leaves it "pending" forever, permanently
		// occupying a seat nobody paid for.
		if markErr := h.orders.MarkFailed(r.Context(), order.ID); markErr != nil {
			h.log.Error("mark order failed", "error", markErr, "orderId", order.ID)
		}
		writeJSONError(w, http.StatusBadGateway, "não foi possível iniciar o pagamento")
		return
	}

	charge, err := h.asaas.CreateCharge(r.Context(), asaas.ChargeParams{
		CustomerID:  customerID,
		BillingType: "PIX",
		ValueCents:  order.TotalCents,
		Description: prod.Title + " — P5 Wellness Club",
		ExternalRef: order.OrderNumber,
	})
	if err != nil {
		h.log.Error("asaas create charge", "error", err)
		if markErr := h.orders.MarkFailed(r.Context(), order.ID); markErr != nil {
			h.log.Error("mark order failed", "error", markErr, "orderId", order.ID)
		}
		writeJSONError(w, http.StatusBadGateway, "não foi possível gerar a cobrança")
		return
	}

	if err := h.orders.SetAsaasPayment(r.Context(), order.ID, "pix", charge.PaymentID); err != nil {
		h.log.Error("set asaas payment", "error", err)
	}

	writeJSON(w, http.StatusCreated, checkoutResponse{
		OrderID: order.ID, OrderNumber: order.OrderNumber, TotalCents: order.TotalCents,
		PixCopyPaste: charge.PixCopyPaste, PixQRImage: charge.PixQRImage,
	})
}

// createFree is the checkout for a R$ 0 product (gratuidade ligada pelo preço no Admin):
// same booking + entitlement steps as a voucher redemption (see voucher_student.go), with
// no Asaas customer or charge — Asaas doesn't accept zero-value charges, and there's
// nothing to collect. The order is still a real order, so it counts as an inscrição in the
// session's booked seats, the reports and the check-in roster.
func (h *CheckoutHandler) createFree(w http.ResponseWriter, r *http.Request, studentID string, prod *product.Product, sessionIDs map[string]string) {
	order, err := h.orders.CreateOrder(r.Context(), studentID, *prod, sessionIDs)
	if err != nil {
		switch {
		case errors.Is(err, postgres.ErrActivityChoiceRequired):
			writeJSONError(w, http.StatusConflict, "escolha uma atividade antes de continuar")
			return
		case errors.Is(err, postgres.ErrSessionRequired):
			writeJSONError(w, http.StatusConflict, "escolha uma turma para: "+activityFromError(err))
			return
		case errors.Is(err, postgres.ErrSessionFull):
			writeJSONError(w, http.StatusConflict, "essa turma acabou de lotar, escolha outra data para: "+activityFromError(err))
			return
		}
		h.log.Error("create free order", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "não foi possível concluir a inscrição")
		return
	}

	if err := h.orders.SetAsaasPayment(r.Context(), order.ID, "free", order.OrderNumber); err != nil {
		h.log.Error("set free payment method", "error", err, "orderId", order.ID)
	}

	result, err := h.orders.MarkPaidAndIssueEntitlements(r.Context(), order.ID, order.OrderNumber, "free", 0, h.qrSecret)
	if err != nil {
		if markErr := h.orders.MarkFailed(r.Context(), order.ID); markErr != nil {
			h.log.Error("mark free order failed", "error", markErr, "orderId", order.ID)
		}
		h.log.Error("issue free entitlements", "error", err, "orderId", order.ID)
		writeJSONError(w, http.StatusInternalServerError, "não foi possível concluir a inscrição")
		return
	}

	if err := sendTicketEmail(r.Context(), h.mailer, h.log, *result); err != nil {
		h.log.Error("send free ticket email", "error", err, "orderId", order.ID)
	}

	writeJSON(w, http.StatusCreated, checkoutResponse{
		OrderID: order.ID, OrderNumber: order.OrderNumber, TotalCents: 0, Free: true,
	})
}

func activityFromError(err error) string {
	msg := err.Error()
	if i := strings.LastIndex(msg, ": "); i >= 0 {
		return msg[i+2:]
	}
	return msg
}

func (h *CheckoutHandler) Status(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "id")
	status, secondsLeft, err := h.orders.PaymentStatus(r.Context(), orderID)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "pedido não encontrado")
		return
	}
	// secondsLeft: tempo até o Pix expirar e a vaga ser liberada (postgres.PendingOrderTTL).
	writeJSON(w, http.StatusOK, map[string]any{"status": status, "secondsLeft": secondsLeft})
}
