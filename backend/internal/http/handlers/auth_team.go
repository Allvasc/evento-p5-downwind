package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"p5wellness/backend/internal/domain/auth"
	"p5wellness/backend/internal/mailer"
	"p5wellness/backend/internal/repository/postgres"
)

type AuthTeamHandler struct {
	team   *postgres.TeamRepository
	resets *postgres.PasswordResetRepository
	issuer *auth.TokenIssuer
	mailer mailer.Sender
	pepper string
	log    *slog.Logger
}

func NewAuthTeamHandler(team *postgres.TeamRepository, resets *postgres.PasswordResetRepository, issuer *auth.TokenIssuer, sender mailer.Sender, pepper string, log *slog.Logger) *AuthTeamHandler {
	return &AuthTeamHandler{team: team, resets: resets, issuer: issuer, mailer: sender, pepper: pepper, log: log}
}

func (h *AuthTeamHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "corpo da requisição inválido")
		return
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))

	member, err := h.team.FindByEmail(r.Context(), email)
	if err != nil || !member.Active || !auth.CheckPassword(member.PasswordHash, req.Password, h.pepper) {
		writeJSONError(w, http.StatusUnauthorized, "credenciais inválidas")
		return
	}

	vendorID := ""
	if member.VendorID != nil {
		vendorID = *member.VendorID
	}

	token, err := h.issuer.IssueTeam(member.ID, member.Role, vendorID)
	if err != nil {
		h.log.Error("issue token", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "não foi possível iniciar sua sessão")
		return
	}
	_ = h.team.UpdateLastLogin(r.Context(), member.ID)

	writeJSON(w, http.StatusOK, map[string]string{
		"token": token, "role": member.Role, "name": member.Name,
		"vendorId": vendorID, "vendorName": member.VendorName,
	})
}

// RequestPasswordReset and ResetPassword mirror the student flow (same
// password_reset_tokens table, generic owner_type) — a team member (including the sole
// admin) who forgets their password isn't otherwise stuck needing server access to run
// cmd/seed-admin again.
func (h *AuthTeamHandler) RequestPasswordReset(w http.ResponseWriter, r *http.Request) {
	var req requestResetRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	email := strings.ToLower(strings.TrimSpace(req.Email))

	defer writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})

	member, err := h.team.FindByEmail(r.Context(), email)
	if err != nil || !member.Active {
		return
	}

	count, err := h.resets.CountRecent(r.Context(), postgres.OwnerTeamMember, member.ID, 15*time.Minute)
	if err != nil || count >= 3 {
		return
	}

	code, err := generateNumericCode(6)
	if err != nil {
		h.log.Error("generate reset code", "error", err)
		return
	}
	codeHash := auth.HashCPF(code, h.pepper)

	if err := h.resets.Create(r.Context(), postgres.OwnerTeamMember, member.ID, codeHash, 15*time.Minute); err != nil {
		h.log.Error("create reset token", "error", err)
		return
	}

	body := fmt.Sprintf("Use o código %s para redefinir sua senha de equipe. Ele expira em 15 minutos.", code)
	if err := h.mailer.Send(r.Context(), member.Email, "Código de recuperação — Painel P5 Wellness Club", "password_reset", body); err != nil {
		h.log.Error("send reset email", "error", err)
	}
}

func (h *AuthTeamHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req resetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "corpo da requisição inválido")
		return
	}
	if len(req.Password) < 8 {
		writeJSONError(w, http.StatusBadRequest, "a nova senha deve ter ao menos 8 caracteres")
		return
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))

	member, err := h.team.FindByEmail(r.Context(), email)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "código inválido ou expirado")
		return
	}

	token, err := h.resets.FindActive(r.Context(), postgres.OwnerTeamMember, member.ID)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "código inválido ou expirado")
		return
	}
	if token.Attempts >= 5 {
		writeJSONError(w, http.StatusBadRequest, "código inválido ou expirado")
		return
	}

	codeHash := auth.HashCPF(req.Code, h.pepper)
	if codeHash != token.CodeHash {
		_ = h.resets.RegisterAttempt(r.Context(), token.ID)
		writeJSONError(w, http.StatusBadRequest, "código inválido ou expirado")
		return
	}

	passwordHash, err := auth.HashPassword(req.Password, h.pepper)
	if err != nil {
		h.log.Error("hash password", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "não foi possível redefinir a senha")
		return
	}
	if err := h.team.UpdatePassword(r.Context(), member.ID, passwordHash); err != nil {
		h.log.Error("update password", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "não foi possível redefinir a senha")
		return
	}
	_ = h.resets.Consume(r.Context(), token.ID)

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
