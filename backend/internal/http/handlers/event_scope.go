package handlers

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"p5wellness/backend/internal/repository/postgres"
)

// EventHeader carries the event the operator picked in the Admin / check-in selector.
// The team API client sends it on every request, so every listing, report and write is
// scoped to that one event and nothing from another event leaks into the screen.
const EventHeader = "X-Event-Id"

// eventScope returns the event a team request is scoped to, or "" when the caller sent
// none (an older cached frontend), in which case listings stay unfiltered as before.
func eventScope(r *http.Request) string {
	v := strings.TrimSpace(r.Header.Get(EventHeader))
	if v == "" {
		v = strings.TrimSpace(r.URL.Query().Get("eventId"))
	}
	if _, err := uuid.Parse(v); err != nil {
		return ""
	}
	return v
}

// eventForWrite is the event a new product, activity or voucher is created in: the one
// selected in the Admin, or the default event for a caller that sent none.
func eventForWrite(r *http.Request) string {
	if id := eventScope(r); id != "" {
		return id
	}
	return postgres.DefaultEventID
}

const errOtherEvent = "este item pertence a outro evento — troque o evento selecionado no topo do painel"

// guardEvent refuses a write on a row of another event than the one the operator has
// selected (e.g. a stale tab still showing event A after switching to event B). It
// writes the error response itself and returns false when the request must stop.
func guardEvent(w http.ResponseWriter, r *http.Request, lookup func(context.Context, string) (string, error), id string) bool {
	scope := eventScope(r)
	if scope == "" {
		return true
	}
	rowEvent, err := lookup(r.Context(), id)
	if errors.Is(err, postgres.ErrNotFound) {
		writeJSONError(w, http.StatusNotFound, "registro não encontrado")
		return false
	}
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "não foi possível verificar o evento")
		return false
	}
	if rowEvent != scope {
		writeJSONError(w, http.StatusConflict, errOtherEvent)
		return false
	}
	return true
}
