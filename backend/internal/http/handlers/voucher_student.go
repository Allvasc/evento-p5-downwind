package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"p5wellness/backend/internal/http/middleware"
	"p5wellness/backend/internal/mailer"
	"p5wellness/backend/internal/repository/postgres"
)

type VoucherStudentHandler struct {
	catalog  *postgres.CatalogRepository
	orders   *postgres.OrderRepository
	vouchers *postgres.VoucherRepository
	mailer   mailer.Sender
	qrSecret string
	log      *slog.Logger
}

func NewVoucherStudentHandler(catalog *postgres.CatalogRepository, orders *postgres.OrderRepository, vouchers *postgres.VoucherRepository, sender mailer.Sender, qrSecret string, log *slog.Logger) *VoucherStudentHandler {
	return &VoucherStudentHandler{catalog: catalog, orders: orders, vouchers: vouchers, mailer: sender, qrSecret: qrSecret, log: log}
}

func normalizeVoucherCode(raw string) string {
	return strings.ToUpper(strings.TrimSpace(raw))
}

// Check is the read-only "is this code valid" lookup shown before the product/date
// picker, so a typo'd or already-used code fails fast instead of after the customer
// has already picked a product and a date.
func (h *VoucherStudentHandler) Check(w http.ResponseWriter, r *http.Request) {
	code := normalizeVoucherCode(chi.URLParam(r, "code"))
	v, err := h.vouchers.FindAvailableByCode(r.Context(), code)
	if err != nil {
		switch {
		case errors.Is(err, postgres.ErrVoucherNotFound):
			writeJSONError(w, http.StatusNotFound, "código de voucher não encontrado")
		case errors.Is(err, postgres.ErrVoucherNotAvailable):
			writeJSONError(w, http.StatusConflict, "esse voucher já foi usado ou cancelado")
		default:
			h.log.Error("check voucher", "error", err)
			writeJSONError(w, http.StatusInternalServerError, "não foi possível verificar o voucher")
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"name": v.Name, "companyName": v.CompanyName})
}

type redeemVoucherRequest struct {
	Code       string            `json:"code"`
	ProductID  string            `json:"productId"`
	SessionIDs map[string]string `json:"sessionIds"`
}

// Redeem plays out the same booking + entitlement-issuing steps as a real checkout
// (CreateOrder, then MarkPaidAndIssueEntitlements — see checkout.go), just without ever
// touching Asaas: the order is paid for with the voucher itself. Claim happens before
// the order exists so a voucher can never be double-spent; if anything after that fails,
// the voucher is released back to available rather than being wasted on a failed attempt.
func (h *VoucherStudentHandler) Redeem(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())

	var req redeemVoucherRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Code) == "" || req.ProductID == "" {
		writeJSONError(w, http.StatusBadRequest, "informe o código do voucher e o produto")
		return
	}
	code := normalizeVoucherCode(req.Code)

	prod, err := h.catalog.GetProductByID(r.Context(), req.ProductID)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "experiência não encontrada")
		return
	}
	// A voucher only covers one activity (the customer's choice, e.g. Yoga ou HYROX) plus
	// breakfast — not the two-class combo, and not a class without breakfast.
	if !prod.ChooseOneActivity || !prod.IncludesBreakfast {
		writeJSONError(w, http.StatusBadRequest, "esse produto não pode ser resgatado com voucher")
		return
	}

	voucherID, err := h.vouchers.Claim(r.Context(), code, claims.UserID())
	if err != nil {
		switch {
		case errors.Is(err, postgres.ErrVoucherNotFound):
			writeJSONError(w, http.StatusNotFound, "código de voucher não encontrado")
		case errors.Is(err, postgres.ErrVoucherNotAvailable):
			writeJSONError(w, http.StatusConflict, "esse voucher já foi usado ou cancelado")
		default:
			h.log.Error("claim voucher", "error", err)
			writeJSONError(w, http.StatusInternalServerError, "não foi possível resgatar o voucher")
		}
		return
	}

	order, err := h.orders.CreateOrder(r.Context(), claims.UserID(), *prod, req.SessionIDs)
	if err != nil {
		if releaseErr := h.vouchers.Release(r.Context(), voucherID); releaseErr != nil {
			h.log.Error("release voucher after failed order", "error", releaseErr, "voucherId", voucherID)
		}
		switch {
		case errors.Is(err, postgres.ErrActivityChoiceRequired):
			writeJSONError(w, http.StatusConflict, "escolha uma atividade antes de continuar")
		case errors.Is(err, postgres.ErrSessionRequired):
			writeJSONError(w, http.StatusConflict, "escolha uma turma para: "+activityFromError(err))
		case errors.Is(err, postgres.ErrSessionFull):
			writeJSONError(w, http.StatusConflict, "essa turma acabou de lotar, escolha outra data para: "+activityFromError(err))
		default:
			h.log.Error("create voucher order", "error", err)
			writeJSONError(w, http.StatusInternalServerError, "não foi possível resgatar o voucher")
		}
		return
	}

	if err := h.orders.SetAsaasPayment(r.Context(), order.ID, "voucher", code); err != nil {
		h.log.Error("set voucher payment method", "error", err, "orderId", order.ID)
	}

	result, err := h.orders.MarkPaidAndIssueEntitlements(r.Context(), order.ID, code, "voucher", 0, h.qrSecret)
	if err != nil {
		if markErr := h.orders.MarkFailed(r.Context(), order.ID); markErr != nil {
			h.log.Error("mark voucher order failed", "error", markErr, "orderId", order.ID)
		}
		if releaseErr := h.vouchers.Release(r.Context(), voucherID); releaseErr != nil {
			h.log.Error("release voucher after failed issuance", "error", releaseErr, "voucherId", voucherID)
		}
		h.log.Error("issue voucher entitlements", "error", err, "orderId", order.ID)
		writeJSONError(w, http.StatusInternalServerError, "não foi possível resgatar o voucher")
		return
	}

	if err := h.vouchers.LinkOrder(r.Context(), voucherID, order.ID); err != nil {
		h.log.Error("link voucher to order", "error", err, "voucherId", voucherID, "orderId", order.ID)
	}
	if err := sendTicketEmail(r.Context(), h.mailer, h.log, *result); err != nil {
		h.log.Error("send voucher ticket email", "error", err, "orderId", order.ID)
	}

	writeJSON(w, http.StatusCreated, map[string]string{"orderId": order.ID, "orderNumber": order.OrderNumber})
}
