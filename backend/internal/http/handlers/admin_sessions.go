package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"p5wellness/backend/internal/repository/postgres"
)

type AdminSessionsHandler struct {
	repo *postgres.SessionRepository
	log  *slog.Logger
}

func NewAdminSessionsHandler(repo *postgres.SessionRepository, log *slog.Logger) *AdminSessionsHandler {
	return &AdminSessionsHandler{repo: repo, log: log}
}

func (h *AdminSessionsHandler) List(w http.ResponseWriter, r *http.Request) {
	sessions, err := h.repo.ListUpcoming(r.Context())
	if err != nil {
		h.log.Error("list sessions", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "não foi possível carregar as turmas")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"sessions": sessions})
}

type createSessionRequest struct {
	ActivityID string `json:"activityId"`
	StartsAt   string `json:"startsAt"` // RFC3339
	EndsAt     string `json:"endsAt"`   // RFC3339
	Capacity   int    `json:"capacity"`
}

func (h *AdminSessionsHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "corpo inválido")
		return
	}
	startsAt, err := time.Parse(time.RFC3339, req.StartsAt)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "data/hora de início inválida")
		return
	}
	endsAt, err := time.Parse(time.RFC3339, req.EndsAt)
	if err != nil || !endsAt.After(startsAt) {
		writeJSONError(w, http.StatusBadRequest, "data/hora de término deve ser depois do início")
		return
	}
	if req.ActivityID == "" || req.Capacity <= 0 {
		writeJSONError(w, http.StatusBadRequest, "atividade e vagas (maior que zero) são obrigatórios")
		return
	}

	id, err := h.repo.Create(r.Context(), req.ActivityID, startsAt, endsAt, req.Capacity)
	if err != nil {
		h.log.Error("create session", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "não foi possível criar a turma")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"id": id})
}

func (h *AdminSessionsHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req createSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "corpo inválido")
		return
	}
	startsAt, err := time.Parse(time.RFC3339, req.StartsAt)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "data/hora de início inválida")
		return
	}
	endsAt, err := time.Parse(time.RFC3339, req.EndsAt)
	if err != nil || !endsAt.After(startsAt) {
		writeJSONError(w, http.StatusBadRequest, "data/hora de término deve ser depois do início")
		return
	}
	if req.Capacity <= 0 {
		writeJSONError(w, http.StatusBadRequest, "vagas deve ser maior que zero")
		return
	}

	booked, err := h.repo.Booked(r.Context(), id)
	if err != nil {
		h.log.Error("booked count", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "não foi possível salvar a turma")
		return
	}
	if req.Capacity < booked {
		writeJSONError(w, http.StatusConflict, fmt.Sprintf("já existem %d vagas reservadas — a capacidade não pode ser menor que isso", booked))
		return
	}

	if err := h.repo.Update(r.Context(), id, startsAt, endsAt, req.Capacity); err != nil {
		h.log.Error("update session", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "não foi possível salvar a turma")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *AdminSessionsHandler) SetStatus(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || (body.Status != "scheduled" && body.Status != "cancelled" && body.Status != "completed") {
		writeJSONError(w, http.StatusBadRequest, "status inválido")
		return
	}
	if err := h.repo.SetStatus(r.Context(), id, body.Status); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "não foi possível atualizar")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
