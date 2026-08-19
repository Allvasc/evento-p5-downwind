package handlers

import (
	"log/slog"
	"net/http"
	"time"

	"p5wellness/backend/internal/repository/postgres"
)

type AdminDashboardHandler struct {
	repo *postgres.DashboardRepository
	log  *slog.Logger
}

func NewAdminDashboardHandler(repo *postgres.DashboardRepository, log *slog.Logger) *AdminDashboardHandler {
	return &AdminDashboardHandler{repo: repo, log: log}
}

func (h *AdminDashboardHandler) Summary(w http.ResponseWriter, r *http.Request) {
	from, to := parseDateRange(r)
	summary, err := h.repo.Summary(r.Context(), from, to)
	if err != nil {
		h.log.Error("dashboard summary", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "não foi possível carregar o resumo")
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func (h *AdminDashboardHandler) SalesByProduct(w http.ResponseWriter, r *http.Request) {
	from, to := parseDateRange(r)
	sales, err := h.repo.SalesByProduct(r.Context(), from, to)
	if err != nil {
		h.log.Error("sales by product", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "não foi possível carregar as vendas")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"sales": sales})
}

// SalesTimeseries defaults to the trailing 14 days when no range is given (the dashboard's
// original behavior); an explicit from/to overrides either end. The span is capped at a
// year so a wide-open range can't force an enormous generate_series.
func (h *AdminDashboardHandler) SalesTimeseries(w http.ResponseWriter, r *http.Request) {
	from, to := parseDateRange(r)
	end := time.Now()
	if to != nil {
		end = *to
	}
	start := end.AddDate(0, 0, -13)
	if from != nil {
		start = *from
	}
	if end.Sub(start) > 366*24*time.Hour {
		start = end.AddDate(-1, 0, -1)
	}
	points, err := h.repo.SalesTimeseries(r.Context(), start, end)
	if err != nil {
		h.log.Error("sales timeseries", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "não foi possível carregar a série histórica")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"points": points})
}
