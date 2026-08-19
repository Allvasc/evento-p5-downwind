package handlers

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/big"
	"net/http"
	"strings"
	"time"

	"p5wellness/backend/internal/domain/auth"
	"p5wellness/backend/internal/mailer"
	"p5wellness/backend/internal/repository/postgres"
)

type AuthStudentHandler struct {
	students *postgres.StudentRepository
	resets   *postgres.PasswordResetRepository
	issuer   *auth.TokenIssuer
	mailer   mailer.Sender
	pepper   string
	log      *slog.Logger
}

func NewAuthStudentHandler(students *postgres.StudentRepository, resets *postgres.PasswordResetRepository, issuer *auth.TokenIssuer, sender mailer.Sender, pepper string, log *slog.Logger) *AuthStudentHandler {
	return &AuthStudentHandler{students: students, resets: resets, issuer: issuer, mailer: sender, pepper: pepper, log: log}
}

type registerRequest struct {
	FullName string `json:"fullName"`
	Phone    string `json:"phone"`
	CPF      string `json:"cpf"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *AuthStudentHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "corpo da requisição inválido")
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.FullName = strings.TrimSpace(req.FullName)

	if req.FullName == "" || req.Email == "" || len(req.Password) < 8 {
		writeJSONError(w, http.StatusBadRequest, "nome, e-mail e senha (mínimo 8 caracteres) são obrigatórios")
		return
	}
	if !strings.Contains(req.Email, "@") {
		writeJSONError(w, http.StatusBadRequest, "e-mail inválido")
		return
	}
	if req.CPF != "" && !auth.IsValidCPF(req.CPF) {
		writeJSONError(w, http.StatusBadRequest, "CPF inválido")
		return
	}

	passwordHash, err := auth.HashPassword(req.Password, h.pepper)
	if err != nil {
		h.log.Error("hash password", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "não foi possível concluir o cadastro")
		return
	}

	params := postgres.CreateStudentParams{
		FullName:      req.FullName,
		Email:         req.Email,
		Phone:         req.Phone,
		PasswordHash:  passwordHash,
		EncryptionKey: h.pepper,
	}
	if req.CPF != "" {
		params.CPF = auth.NormalizeCPF(req.CPF)
		params.CPFHash = auth.HashCPF(req.CPF, h.pepper)
		params.CPFLast4 = auth.LastFour(req.CPF)
	}

	student, err := h.students.Create(r.Context(), params)
	if err != nil {
		if err == postgres.ErrConflict {
			writeJSONError(w, http.StatusConflict, "já existe um cadastro com esse e-mail ou CPF")
			return
		}
		h.log.Error("create student", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "não foi possível concluir o cadastro")
		return
	}

	token, err := h.issuer.Issue(student.ID, auth.SubjectStudent, "student")
	if err != nil {
		h.log.Error("issue token", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "cadastro criado, mas não foi possível iniciar sua sessão")
		return
	}

	// Best-effort: a welcome e-mail failing must never block registration itself.
	if err := h.mailer.SendWelcome(r.Context(), student.Email, student.FullName); err != nil {
		h.log.Error("send welcome email", "error", err)
	}

	writeJSON(w, http.StatusCreated, map[string]string{"token": token})
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *AuthStudentHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "corpo da requisição inválido")
		return
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))

	student, err := h.students.FindByEmail(r.Context(), email)
	if err != nil || !auth.CheckPassword(student.PasswordHash, req.Password, h.pepper) {
		writeJSONError(w, http.StatusUnauthorized, "e-mail ou senha incorretos")
		return
	}

	token, err := h.issuer.Issue(student.ID, auth.SubjectStudent, "student")
	if err != nil {
		h.log.Error("issue token", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "não foi possível iniciar sua sessão")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"token": token})
}

type requestResetRequest struct {
	Email string `json:"email"`
}

func (h *AuthStudentHandler) RequestPasswordReset(w http.ResponseWriter, r *http.Request) {
	var req requestResetRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	email := strings.ToLower(strings.TrimSpace(req.Email))

	// Always respond success regardless of whether the e-mail exists, to avoid
	// leaking which e-mails are registered.
	defer writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})

	student, err := h.students.FindByEmail(r.Context(), email)
	if err != nil {
		return
	}

	count, err := h.resets.CountRecent(r.Context(), postgres.OwnerStudent, student.ID, 15*time.Minute)
	if err != nil || count >= 3 {
		return
	}

	code, err := generateNumericCode(6)
	if err != nil {
		h.log.Error("generate reset code", "error", err)
		return
	}
	codeHash := auth.HashCPF(code, h.pepper) // sha256+pepper, reused helper works for any short secret

	if err := h.resets.Create(r.Context(), postgres.OwnerStudent, student.ID, codeHash, 15*time.Minute); err != nil {
		h.log.Error("create reset token", "error", err)
		return
	}

	body := fmt.Sprintf("Use o código %s para redefinir sua senha. Ele expira em 15 minutos.", code)
	if err := h.mailer.Send(r.Context(), student.Email, "Código de recuperação — P5 Wellness Club", "password_reset", body); err != nil {
		h.log.Error("send reset email", "error", err)
	}
}

type resetPasswordRequest struct {
	Email    string `json:"email"`
	Code     string `json:"code"`
	Password string `json:"password"`
}

func (h *AuthStudentHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
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

	student, err := h.students.FindByEmail(r.Context(), email)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "código inválido ou expirado")
		return
	}

	token, err := h.resets.FindActive(r.Context(), postgres.OwnerStudent, student.ID)
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
	if err := h.students.UpdatePassword(r.Context(), student.ID, passwordHash); err != nil {
		h.log.Error("update password", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "não foi possível redefinir a senha")
		return
	}
	_ = h.resets.Consume(r.Context(), token.ID)

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func generateNumericCode(digits int) (string, error) {
	var b strings.Builder
	for i := 0; i < digits; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&b, "%d", n.Int64())
	}
	return b.String(), nil
}
