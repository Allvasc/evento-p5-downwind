package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"regexp"
	"strings"

	"github.com/go-chi/chi/v5"

	"p5wellness/backend/internal/repository/postgres"
)

type EventsHandler struct {
	events *postgres.EventRepository
	log    *slog.Logger
}

func NewEventsHandler(events *postgres.EventRepository, log *slog.Logger) *EventsHandler {
	return &EventsHandler{events: events, log: log}
}

// PublicEvent is what the /<slug> page needs — no internal flags beyond isDefault (the
// default event's page is "/", so its slug page redirects there).
type PublicEvent struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Slug          string `json:"slug"`
	IsDefault     bool   `json:"isDefault"`
	Headline      string `json:"headline"`
	Subtitle      string `json:"subtitle"`
	DateLabel     string `json:"dateLabel"`
	LocationLabel string `json:"locationLabel"`
	Description   string `json:"description"`
	BannerURL     string `json:"bannerUrl"`
	Includes      string `json:"includes"`
	Schedule      string `json:"schedule"`
}

// Public serves GET /public/events/{slug}: a published event's page content.
func (h *EventsHandler) Public(w http.ResponseWriter, r *http.Request) {
	e, err := h.events.GetActiveBySlug(r.Context(), chi.URLParam(r, "slug"))
	if errors.Is(err, postgres.ErrNotFound) {
		writeJSONError(w, http.StatusNotFound, "evento não encontrado")
		return
	}
	if err != nil {
		h.log.Error("public event", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "não foi possível carregar o evento")
		return
	}
	writeJSON(w, http.StatusOK, PublicEvent{
		ID: e.ID, Name: e.Name, Slug: e.Slug, IsDefault: e.IsDefault,
		Headline: e.Headline, Subtitle: e.Subtitle, DateLabel: e.DateLabel, LocationLabel: e.LocationLabel,
		Description: e.Description, BannerURL: e.BannerURL, Includes: e.Includes, Schedule: e.Schedule,
	})
}

// teamEvent is the slim row behind the event selector of the Admin and the check-in
// terminal, readable by every team role (the team is shared by all events).
type teamEvent struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	IsDefault bool   `json:"isDefault"`
	Active    bool   `json:"active"`
}

// TeamList serves GET /team/events.
func (h *EventsHandler) TeamList(w http.ResponseWriter, r *http.Request) {
	list, err := h.events.List(r.Context())
	if err != nil {
		h.log.Error("list events", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "não foi possível carregar os eventos")
		return
	}
	out := make([]teamEvent, 0, len(list))
	for _, e := range list {
		out = append(out, teamEvent{ID: e.ID, Name: e.Name, Slug: e.Slug, IsDefault: e.IsDefault, Active: e.Active})
	}
	writeJSON(w, http.StatusOK, map[string]any{"events": out})
}

// AdminList serves GET /admin/events: every event with its full page content.
func (h *EventsHandler) AdminList(w http.ResponseWriter, r *http.Request) {
	list, err := h.events.List(r.Context())
	if err != nil {
		h.log.Error("list events", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "não foi possível carregar os eventos")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"events": list})
}

type eventRequest struct {
	Name          string `json:"name"`
	Slug          string `json:"slug"`
	Headline      string `json:"headline"`
	Subtitle      string `json:"subtitle"`
	DateLabel     string `json:"dateLabel"`
	LocationLabel string `json:"locationLabel"`
	Description   string `json:"description"`
	BannerURL     string `json:"bannerUrl"`
	Includes      string `json:"includes"`
	Schedule      string `json:"schedule"`
}

var eventSlugPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// reservedEventSlugs are the site's own pages and paths: an event at one of these would
// either be unreachable or hijack an existing page.
var reservedEventSlugs = map[string]bool{
	"comprar": true, "entrar": true, "acesso-admin": true, "portal": true, "voucher": true,
	"perfil": true, "check-in": true, "checkin": true, "admin": true, "api": true,
	"healthz": true, "assets": true, "evento": true, "eventos": true, "index": true,
	"favicon": true, "robots": true, "sitemap": true, "login": true, "cadastro": true,
}

// normalize trims every field and checks name, slug and banner URL, returning a
// user-facing error message ("" when valid).
func (req *eventRequest) normalize() string {
	req.Name = strings.TrimSpace(req.Name)
	req.Slug = strings.ToLower(strings.TrimSpace(req.Slug))
	req.Headline = strings.TrimSpace(req.Headline)
	req.Subtitle = strings.TrimSpace(req.Subtitle)
	req.DateLabel = strings.TrimSpace(req.DateLabel)
	req.LocationLabel = strings.TrimSpace(req.LocationLabel)
	req.Description = strings.TrimSpace(req.Description)
	req.BannerURL = strings.TrimSpace(req.BannerURL)
	req.Includes = strings.TrimSpace(req.Includes)
	req.Schedule = strings.TrimSpace(req.Schedule)

	if req.Name == "" {
		return "informe o nome do evento"
	}
	if !eventSlugPattern.MatchString(req.Slug) || len(req.Slug) > 60 {
		return "o endereço da página deve ter só letras minúsculas, números e hífens (ex: downwind-carnaval)"
	}
	if reservedEventSlugs[req.Slug] {
		return "esse endereço já é usado por uma página do site, escolha outro"
	}
	if req.BannerURL != "" && !strings.HasPrefix(req.BannerURL, "https://") && !strings.HasPrefix(req.BannerURL, "/") {
		return "o banner deve ser um link https:// ou um arquivo do site (começando com /)"
	}
	return ""
}

func (req eventRequest) input() postgres.EventInput {
	return postgres.EventInput{
		Name: req.Name, Slug: req.Slug, Headline: req.Headline, Subtitle: req.Subtitle,
		DateLabel: req.DateLabel, LocationLabel: req.LocationLabel, Description: req.Description,
		BannerURL: req.BannerURL, Includes: req.Includes, Schedule: req.Schedule,
	}
}

// Create serves POST /admin/events. A new event starts unpublished.
func (h *EventsHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req eventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "corpo inválido")
		return
	}
	if msg := req.normalize(); msg != "" {
		writeJSONError(w, http.StatusBadRequest, msg)
		return
	}
	if !h.nameAvailable(w, r, req.Name, "") {
		return
	}
	id, err := h.events.Create(r.Context(), req.input())
	if errors.Is(err, postgres.ErrEventSlugTaken) {
		writeJSONError(w, http.StatusConflict, "já existe um evento com esse endereço")
		return
	}
	if err != nil {
		h.log.Error("create event", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "não foi possível criar o evento")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"id": id})
}

// Update serves PUT /admin/events/{id}. The main event's slug is not editable: its page
// is "/" and its slug only exists to redirect there.
func (h *EventsHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req eventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "corpo inválido")
		return
	}
	current, err := h.events.GetByID(r.Context(), id)
	if errors.Is(err, postgres.ErrNotFound) {
		writeJSONError(w, http.StatusNotFound, "evento não encontrado")
		return
	}
	if err != nil {
		h.log.Error("get event", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "não foi possível salvar o evento")
		return
	}
	if current.IsDefault {
		req.Slug = current.Slug
	}
	if msg := req.normalize(); msg != "" {
		writeJSONError(w, http.StatusBadRequest, msg)
		return
	}
	if !h.nameAvailable(w, r, req.Name, id) {
		return
	}
	err = h.events.Update(r.Context(), id, req.input())
	if errors.Is(err, postgres.ErrEventSlugTaken) {
		writeJSONError(w, http.StatusConflict, "já existe um evento com esse endereço")
		return
	}
	if err != nil {
		h.log.Error("update event", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "não foi possível salvar o evento")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// nameAvailable refuses a name another event already uses, writing the error itself.
func (h *EventsHandler) nameAvailable(w http.ResponseWriter, r *http.Request, name, exceptID string) bool {
	taken, err := h.events.NameTaken(r.Context(), name, exceptID)
	if err != nil {
		h.log.Error("check event name", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "não foi possível salvar o evento")
		return false
	}
	if taken {
		writeJSONError(w, http.StatusConflict, "já existe um evento com esse nome — use um nome diferente para não confundir os eventos")
		return false
	}
	return true
}

// SetActive serves PATCH /admin/events/{id}/active (publish / unpublish the page).
func (h *EventsHandler) SetActive(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		Active bool `json:"active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "corpo inválido")
		return
	}
	err := h.events.SetActive(r.Context(), id, body.Active)
	switch {
	case errors.Is(err, postgres.ErrDefaultEventLocked):
		writeJSONError(w, http.StatusConflict, "o evento principal não pode ser desativado")
	case errors.Is(err, postgres.ErrNotFound):
		writeJSONError(w, http.StatusNotFound, "evento não encontrado")
	case err != nil:
		h.log.Error("set event active", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "não foi possível atualizar o evento")
	default:
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}
