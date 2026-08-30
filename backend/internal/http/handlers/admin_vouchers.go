package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"p5wellness/backend/internal/http/middleware"
	"p5wellness/backend/internal/repository/postgres"
)

type AdminVouchersHandler struct {
	vouchers *postgres.VoucherRepository
	log      *slog.Logger
}

func NewAdminVouchersHandler(vouchers *postgres.VoucherRepository, log *slog.Logger) *AdminVouchersHandler {
	return &AdminVouchersHandler{vouchers: vouchers, log: log}
}

func (h *AdminVouchersHandler) List(w http.ResponseWriter, r *http.Request) {
	list, err := h.vouchers.List(r.Context())
	if err != nil {
		h.log.Error("list vouchers", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "não foi possível carregar os vouchers")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"vouchers": list})
}

type createVoucherRequest struct {
	Name        string `json:"name"`
	CompanyName string `json:"companyName"`
	// Count is how many independent codes to generate in one go — a partner company
	// handing courtesies to a whole team needs a batch, not one-by-one clicks. Defaults
	// to 1 when omitted or non-positive; capped well above any realistic campaign size
	// so a typo (e.g. an extra zero) can't mint thousands of codes by accident.
	Count int `json:"count"`
}

const maxVoucherBatch = 500

func (h *AdminVouchersHandler) Create(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())

	var req createVoucherRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.CompanyName) == "" {
		writeJSONError(w, http.StatusBadRequest, "informe o nome do voucher e o nome da empresa")
		return
	}
	count := req.Count
	if count < 1 {
		count = 1
	}
	if count > maxVoucherBatch {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("no máximo %d vouchers por vez", maxVoucherBatch))
		return
	}

	created, err := h.vouchers.CreateBatch(r.Context(), strings.TrimSpace(req.Name), strings.TrimSpace(req.CompanyName), claims.UserID(), count)
	if err != nil {
		h.log.Error("create voucher batch", "error", err, "createdSoFar", len(created))
		if len(created) == 0 {
			writeJSONError(w, http.StatusInternalServerError, "não foi possível criar o voucher")
			return
		}
		// Partial failure mid-batch (rare — code-generation exhausted retries): still
		// return what succeeded rather than losing already-committed codes.
		writeJSON(w, http.StatusCreated, map[string]any{"vouchers": created, "warning": "alguns códigos não puderam ser gerados, veja a lista"})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"vouchers": created})
}

func (h *AdminVouchersHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.vouchers.Cancel(r.Context(), id); err != nil {
		switch {
		case errors.Is(err, postgres.ErrVoucherNotFound):
			writeJSONError(w, http.StatusNotFound, "voucher não encontrado")
		case errors.Is(err, postgres.ErrVoucherHasRedemption):
			writeJSONError(w, http.StatusConflict, "esse voucher já foi resgatado, não é possível cancelar")
		default:
			h.log.Error("cancel voucher", "error", err)
			writeJSONError(w, http.StatusInternalServerError, "não foi possível cancelar o voucher")
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
