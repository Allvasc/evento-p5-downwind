package handlers

import (
	"encoding/csv"
	"fmt"
	"log/slog"
	"net/http"

	"p5wellness/backend/internal/repository/postgres"
)

type AdminReportsHandler struct {
	repo          *postgres.AdminReportsRepository
	encryptionKey string
	log           *slog.Logger
}

func NewAdminReportsHandler(repo *postgres.AdminReportsRepository, encryptionKey string, log *slog.Logger) *AdminReportsHandler {
	return &AdminReportsHandler{repo: repo, encryptionKey: encryptionKey, log: log}
}

func (h *AdminReportsHandler) Sessions(w http.ResponseWriter, r *http.Request) {
	from, to := parseDateRange(r)
	list, err := h.repo.SessionsWithAttendees(r.Context(), from, to)
	if err != nil {
		h.log.Error("sessions report", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "não foi possível carregar o relatório de turmas")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"sessions": list})
}

func (h *AdminReportsHandler) Products(w http.ResponseWriter, r *http.Request) {
	from, to := parseDateRange(r)
	list, err := h.repo.ProductsWithBuyers(r.Context(), from, to)
	if err != nil {
		h.log.Error("products report", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "não foi possível carregar o relatório de produtos")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"products": list})
}

func (h *AdminReportsHandler) Activities(w http.ResponseWriter, r *http.Request) {
	from, to := parseDateRange(r)
	list, err := h.repo.ByActivity(r.Context(), from, to)
	if err != nil {
		h.log.Error("activities report", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "não foi possível carregar o relatório por atividade")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"activities": list})
}

// Attendees backs the "Lista de presença" screen: every paid benefit due on the event
// day with the buyer's phone, e-mail, CPF and purchase date.
func (h *AdminReportsHandler) Attendees(w http.ResponseWriter, r *http.Request) {
	from, to := parseDateRange(r)
	list, err := h.repo.EventAttendees(r.Context(), from, to, h.encryptionKey)
	if err != nil {
		h.log.Error("attendees report", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "não foi possível carregar a lista de presença")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"attendees": list})
}

// AttendeesCSV is the same list as a spreadsheet download, meant to be printed and
// carried on a clipboard: a leading "Presença" column holds an empty checkbox (☐) for
// ticking with a pen, and "Check-in no sistema" shows whether the QR was already
// validated digitally.
func (h *AdminReportsHandler) AttendeesCSV(w http.ResponseWriter, r *http.Request) {
	from, to := parseDateRange(r)
	list, err := h.repo.EventAttendees(r.Context(), from, to, h.encryptionKey)
	if err != nil {
		h.log.Error("attendees csv report", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "não foi possível gerar a lista de presença")
		return
	}

	filename := "lista-presenca"
	if v := r.URL.Query().Get("from"); v != "" {
		filename += "-" + v
	}
	filename += ".csv"

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	// BOM para o Excel abrir os acentos corretamente.
	_, _ = w.Write([]byte{0xEF, 0xBB, 0xBF})

	cw := csv.NewWriter(w)
	cw.UseCRLF = true
	_ = cw.Write([]string{
		"Presença", "Nome", "Telefone", "E-mail", "CPF", "Data da compra",
		"Pedido", "Ingresso", "Turma", "Dia do evento", "Setor", "Check-in no sistema",
	})
	for _, a := range list {
		checkedIn := "Não"
		if a.CheckedIn {
			checkedIn = "Sim"
			if a.CheckedInAt != "" {
				checkedIn = "Sim — " + a.CheckedInAt
			}
		}
		_ = cw.Write([]string{
			"☐", a.FullName, a.Phone, a.Email, a.CPF, a.PurchasedAt,
			a.OrderNumber, a.Benefit, a.SessionAt, a.EventDate, a.VendorName, checkedIn,
		})
	}
	cw.Flush()
	if err := cw.Error(); err != nil {
		h.log.Error("attendees csv write", "error", err)
	}
}
