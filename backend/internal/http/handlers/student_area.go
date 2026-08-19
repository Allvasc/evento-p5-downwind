package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	qrcodelib "github.com/skip2/go-qrcode"

	"p5wellness/backend/internal/domain/auth"
	"p5wellness/backend/internal/http/middleware"
	"p5wellness/backend/internal/mailer"
	"p5wellness/backend/internal/repository/postgres"
)

type StudentAreaHandler struct {
	students *postgres.StudentRepository
	area     *postgres.StudentAreaRepository
	orders   *postgres.OrderRepository
	mailer   mailer.Sender
	pepper   string
	log      *slog.Logger
}

func NewStudentAreaHandler(students *postgres.StudentRepository, area *postgres.StudentAreaRepository, orders *postgres.OrderRepository, sender mailer.Sender, pepper string, log *slog.Logger) *StudentAreaHandler {
	return &StudentAreaHandler{students: students, area: area, orders: orders, mailer: sender, pepper: pepper, log: log}
}

func (h *StudentAreaHandler) ListOrders(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	orders, err := h.area.ListOrders(r.Context(), claims.UserID())
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "não foi possível carregar seus pedidos")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"orders": orders})
}

func (h *StudentAreaHandler) ListTickets(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	tickets, err := h.area.ListTickets(r.Context(), claims.UserID())
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "não foi possível carregar seus benefícios")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"tickets": tickets})
}

func (h *StudentAreaHandler) TicketQRCode(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	ticketID := chi.URLParam(r, "id")

	token, err := h.area.TokenForOwnedTicket(r.Context(), claims.UserID(), ticketID)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "benefício não encontrado")
		return
	}

	png, err := qrcodelib.Encode(token, qrcodelib.Medium, 320)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "não foi possível gerar o QR Code")
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "private, max-age=60")
	_, _ = w.Write(png)
}

func (h *StudentAreaHandler) Me(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	student, err := h.students.FindByID(r.Context(), claims.UserID())
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "aluno não encontrado")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":       student.ID,
		"fullName": student.FullName,
		"email":    student.Email,
		"phone":    student.Phone,
		"cpfLast4": student.CPFLast4,
	})
}

type updateCPFRequest struct {
	CPF string `json:"cpf"`
}

// UpdateCPF lets a student add or correct their CPF after registration — necessary
// because checkout blocks Pix/card payment without one, but registration itself never
// required it.
func (h *StudentAreaHandler) UpdateCPF(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())

	var req updateCPFRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "corpo inválido")
		return
	}
	if !auth.IsValidCPF(req.CPF) || auth.NormalizeCPF(req.CPF) == "" {
		writeJSONError(w, http.StatusBadRequest, "CPF inválido")
		return
	}

	cpf := auth.NormalizeCPF(req.CPF)
	cpfHash := auth.HashCPF(req.CPF, h.pepper)
	cpfLast4 := auth.LastFour(req.CPF)

	if err := h.students.UpdateCPF(r.Context(), claims.UserID(), cpf, cpfHash, cpfLast4, h.pepper); err != nil {
		if err == postgres.ErrConflict {
			writeJSONError(w, http.StatusConflict, "esse CPF já está associado a outro cadastro")
			return
		}
		h.log.Error("update cpf", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "não foi possível atualizar o CPF")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"cpfLast4": cpfLast4})
}

// ResendEmail re-sends the order confirmation e-mail (with every QR ticket already
// issued for it) — a self-service fix for the common "I didn't get the e-mail" case
// without needing to contact support.
func (h *StudentAreaHandler) ResendEmail(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	orderID := chi.URLParam(r, "id")

	detail, err := h.orders.GetPaidOrderDetail(r.Context(), orderID)
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
	if detail.StudentID != claims.UserID() {
		writeJSONError(w, http.StatusNotFound, "pedido não encontrado")
		return
	}

	if err := sendTicketEmail(r.Context(), h.mailer, h.log, *detail); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "não foi possível reenviar o e-mail")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
