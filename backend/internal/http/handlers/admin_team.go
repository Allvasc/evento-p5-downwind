package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"p5wellness/backend/internal/domain/auth"
	"p5wellness/backend/internal/repository/postgres"
)

type AdminTeamHandler struct {
	team   *postgres.TeamRepository
	pepper string
	log    *slog.Logger
}

func NewAdminTeamHandler(team *postgres.TeamRepository, pepper string, log *slog.Logger) *AdminTeamHandler {
	return &AdminTeamHandler{team: team, pepper: pepper, log: log}
}

func (h *AdminTeamHandler) List(w http.ResponseWriter, r *http.Request) {
	members, err := h.team.ListAll(r.Context())
	if err != nil {
		h.log.Error("list team", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "não foi possível carregar a equipe")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"members": members})
}

type createTeamMemberRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Password string `json:"password"`
	Role     string `json:"role"`
	VendorID string `json:"vendorId"`
}

func (h *AdminTeamHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createTeamMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "corpo inválido")
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if req.Name == "" || req.Email == "" || len(req.Password) < 8 {
		writeJSONError(w, http.StatusBadRequest, "nome, e-mail e senha (mínimo 8 caracteres) são obrigatórios")
		return
	}
	if req.Role != "admin" && req.Role != "staff" && req.Role != "reports" {
		writeJSONError(w, http.StatusBadRequest, "role deve ser admin, staff ou reports")
		return
	}
	// Funcionário (staff) só valida QR do próprio setor — precisa de um setor definido.
	// Admin (P5) enxerga/valida todos os setores, então fica sem vínculo.
	var vendorID *string
	if req.Role == "staff" {
		if req.VendorID == "" {
			writeJSONError(w, http.StatusBadRequest, "selecione o setor deste funcionário")
			return
		}
		vendorID = &req.VendorID
	}

	hash, err := auth.HashPassword(req.Password, h.pepper)
	if err != nil {
		h.log.Error("hash password", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "não foi possível criar o acesso")
		return
	}

	member, err := h.team.Create(r.Context(), postgres.CreateTeamMemberParams{
		Name: req.Name, Email: req.Email, Phone: req.Phone, PasswordHash: hash, Role: req.Role, VendorID: vendorID,
	})
	if err != nil {
		if err == postgres.ErrConflict {
			writeJSONError(w, http.StatusConflict, "já existe uma conta com esse e-mail")
			return
		}
		h.log.Error("create team member", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "não foi possível criar o acesso")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"id": member.ID})
}

func (h *AdminTeamHandler) SetActive(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		Active bool `json:"active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "corpo inválido")
		return
	}
	if err := h.team.SetActive(r.Context(), id, body.Active); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "não foi possível atualizar")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
