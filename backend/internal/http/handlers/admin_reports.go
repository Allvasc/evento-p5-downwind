package handlers

import (
	"log/slog"
	"net/http"

	"p5wellness/backend/internal/repository/postgres"
)

type AdminReportsHandler struct {
	repo *postgres.AdminReportsRepository
	log  *slog.Logger
}

func NewAdminReportsHandler(repo *postgres.AdminReportsRepository, log *slog.Logger) *AdminReportsHandler {
	return &AdminReportsHandler{repo: repo, log: log}
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
