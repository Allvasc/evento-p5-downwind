package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"unicode"

	"p5wellness/backend/internal/http/middleware"
)

// errEmergencyContactRequired is the message shown when an account created before the
// emergency contact became mandatory tries to buy or redeem a voucher without it.
const errEmergencyContactRequired = "informe um contato de emergência (nome e telefone) antes de continuar"

// normalizeEmergencyContact trims and validates the emergency contact sent by the
// customer. It returns a user-facing error message ("" when valid).
func normalizeEmergencyContact(name, phone string) (string, string, string) {
	name = strings.TrimSpace(name)
	phone = strings.TrimSpace(phone)
	if len([]rune(name)) < 2 {
		return "", "", "informe o nome do contato de emergência"
	}
	digits := 0
	for _, r := range phone {
		if unicode.IsDigit(r) {
			digits++
		}
	}
	if digits < 10 || digits > 13 || len(phone) > 20 {
		return "", "", "informe um telefone válido para o contato de emergência, com DDD"
	}
	return name, phone, ""
}

type updateEmergencyContactRequest struct {
	Name  string `json:"name"`
	Phone string `json:"phone"`
}

// UpdateEmergencyContact lets a student fill in (or correct) their emergency contact —
// required before any purchase or voucher redemption.
func (h *StudentAreaHandler) UpdateEmergencyContact(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())

	var req updateEmergencyContactRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "corpo inválido")
		return
	}
	name, phone, msg := normalizeEmergencyContact(req.Name, req.Phone)
	if msg != "" {
		writeJSONError(w, http.StatusBadRequest, msg)
		return
	}
	if err := h.students.UpdateEmergencyContact(r.Context(), claims.UserID(), name, phone); err != nil {
		h.log.Error("update emergency contact", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "não foi possível salvar o contato de emergência")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"emergencyContactName": name, "emergencyContactPhone": phone})
}
