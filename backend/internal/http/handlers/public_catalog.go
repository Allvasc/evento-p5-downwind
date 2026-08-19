package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"p5wellness/backend/internal/repository/postgres"
)

type PublicCatalogHandler struct {
	catalog  *postgres.CatalogRepository
	sessions *postgres.SessionRepository
	log      *slog.Logger
}

func NewPublicCatalogHandler(catalog *postgres.CatalogRepository, sessions *postgres.SessionRepository, log *slog.Logger) *PublicCatalogHandler {
	return &PublicCatalogHandler{catalog: catalog, sessions: sessions, log: log}
}

func (h *PublicCatalogHandler) ListProducts(w http.ResponseWriter, r *http.Request) {
	products, err := h.catalog.ListPublicProducts(r.Context())
	if err != nil {
		h.log.Error("list public products", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "não foi possível carregar o catálogo")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"products": products})
}

// ListActivitySessions returns the upcoming, open-seat sessions for one activity — the
// "Selecione sua data" cards shown at checkout.
func (h *PublicCatalogHandler) ListActivitySessions(w http.ResponseWriter, r *http.Request) {
	activityID := chi.URLParam(r, "activityId")
	sessions, err := h.sessions.AvailableForActivity(r.Context(), activityID)
	if err != nil {
		h.log.Error("list activity sessions", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "não foi possível carregar as turmas")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"sessions": sessions})
}

// NextSessions is what the landing page shows before checkout: the soonest upcoming
// date for each active activity, so a visitor knows when the next event is without
// picking a product first.
func (h *PublicCatalogHandler) NextSessions(w http.ResponseWriter, r *http.Request) {
	sessions, err := h.sessions.NextPerActivity(r.Context())
	if err != nil {
		h.log.Error("list next sessions", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "não foi possível carregar as próximas datas")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"sessions": sessions})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// parseDateRange reads optional ?from=YYYY-MM-DD&to=YYYY-MM-DD query params, shared by
// every admin listing/report that supports scoping to a date range. Either or both may be
// omitted, leaving that bound open; malformed values are ignored (treated as omitted)
// rather than erroring out, since a date filter here is a convenience, not a hard
// requirement.
func parseDateRange(r *http.Request) (from, to *time.Time) {
	if v := r.URL.Query().Get("from"); v != "" {
		if parsed, err := time.Parse("2006-01-02", v); err == nil {
			from = &parsed
		}
	}
	if v := r.URL.Query().Get("to"); v != "" {
		if parsed, err := time.Parse("2006-01-02", v); err == nil {
			to = &parsed
		}
	}
	return from, to
}
