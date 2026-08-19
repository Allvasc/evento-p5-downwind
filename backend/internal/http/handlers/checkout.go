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
	"p5wellness/backend/internal/repository/postgres"
)

type CheckoutHandler struct {
	catalog       *postgres.CatalogRepository
	orders        *postgres.OrderRepository
	students      *postgres.StudentRepository
	asaas         asaas.Client
	encryptionKey string
	log           *slog.Logger
}

func NewCheckoutHandler(catalog *postgres.CatalogRepository, orders *postgres.OrderRepository, students *postgres.StudentRepository, asaasClient asaas.Client, encryptionKey string, log *slog.Logger) *CheckoutHandler {
	return &CheckoutHandler{catalog: catalog, orders: orders, students: students, asaas: asaasClient, encryptionKey: encryptionKey, log: log}
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

func activityFromError(err error) string {
	msg := err.Error()
	if i := strings.LastIndex(msg, ": "); i >= 0 {
		return msg[i+2:]
	}
	return msg
}

func (h *CheckoutHandler) Status(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "id")
	_, status, err := h.orders.FindByID(r.Context(), orderID)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "pedido não encontrado")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": status})
}
