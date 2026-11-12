package handlers

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/mail"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"p5wellness/backend/internal/mailer"
	"p5wellness/backend/internal/repository/postgres"
)

// Inscrição pelo formulário da página do evento (modo "form", ex.: CORE Demo Day):
// sem conta, sem pagamento. A equipe acompanha pela aba Inscrições do Admin.

type EventRegistrationsHandler struct {
	events *postgres.EventRepository
	regs   *postgres.EventRegistrationRepository
	mailer mailer.Sender
	log    *slog.Logger
}

func NewEventRegistrationsHandler(events *postgres.EventRepository, regs *postgres.EventRegistrationRepository, sender mailer.Sender, log *slog.Logger) *EventRegistrationsHandler {
	return &EventRegistrationsHandler{events: events, regs: regs, mailer: sender, log: log}
}

type registrationRequest struct {
	FullName      string `json:"fullName"`
	WhatsApp      string `json:"whatsapp"`
	Email         string `json:"email"`
	PracticeLevel string `json:"practiceLevel"`
	Interest      string `json:"interest"`
	WeightKg      *int   `json:"weightKg"`
	AcceptRules   bool   `json:"acceptRules"`
	Consent       bool   `json:"consent"`
	// Antispam: Website is a hidden field real visitors never fill; ElapsedMs is how
	// long the form was open before submitting (bots post instantly).
	Website   string `json:"website"`
	ElapsedMs int    `json:"elapsedMs"`
}

var practiceLevels = map[string]string{"iniciante": "Iniciante", "intermediario": "Intermediário", "avancado": "Avançado"}
var interests = map[string]string{"kite": "Kite", "prancha": "Prancha", "ambos": "Kite e prancha"}
var nonDigits = regexp.MustCompile(`\D`)

// minFormMs: nobody fills name, WhatsApp, e-mail, two choices and two checkboxes in
// under 2 seconds — a faster submit is a bot.
const minFormMs = 2000

// consentText is what the page shows next to the consent checkbox; the exact wording
// accepted is stored with each sign-up (LGPD: prova do consentimento).
func consentText(eventName string) string {
	return fmt.Sprintf("Autorizo o uso do meu nome, WhatsApp, e-mail, nível de prática, interesse e peso (se informado) "+
		"pela P5 Kite House exclusivamente para organizar minha inscrição e participação no %s e para contato sobre este evento. "+
		"Os dados ficam restritos à equipe organizadora e posso pedir a exclusão a qualquer momento.", eventName)
}

// validate normalizes the request in place and returns a user-facing error ("" = ok).
func (req *registrationRequest) validate() string {
	req.FullName = strings.Join(strings.Fields(req.FullName), " ")
	req.WhatsApp = nonDigits.ReplaceAllString(req.WhatsApp, "")
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.PracticeLevel = strings.TrimSpace(req.PracticeLevel)
	req.Interest = strings.TrimSpace(req.Interest)

	switch {
	case len(req.FullName) < 5 || len(strings.Fields(req.FullName)) < 2 || len(req.FullName) > 120:
		return "informe seu nome completo"
	case len(req.WhatsApp) < 10 || len(req.WhatsApp) > 13:
		return "informe um WhatsApp válido, com DDD"
	case len(req.Email) > 254:
		return "informe um e-mail válido"
	case practiceLevels[req.PracticeLevel] == "":
		return "escolha o seu nível de prática"
	case interests[req.Interest] == "":
		return "escolha o seu interesse: kite, prancha ou ambos"
	case req.WeightKg != nil && (*req.WeightKg < 20 || *req.WeightKg > 250):
		return "informe o peso em kg (entre 20 e 250) ou deixe em branco"
	case !req.AcceptRules:
		return "é preciso aceitar as regras do evento"
	case !req.Consent:
		return "é preciso autorizar o uso dos dados para a inscrição"
	}
	if addr, err := mail.ParseAddress(req.Email); err != nil || addr.Address != req.Email || !strings.Contains(req.Email[strings.LastIndex(req.Email, "@")+1:], ".") {
		return "informe um e-mail válido"
	}
	return ""
}

// Register serves POST /public/events/{slug}/registrations.
func (h *EventRegistrationsHandler) Register(w http.ResponseWriter, r *http.Request) {
	event, err := h.events.GetActiveBySlug(r.Context(), chi.URLParam(r, "slug"))
	if errors.Is(err, postgres.ErrNotFound) {
		writeJSONError(w, http.StatusNotFound, "evento não encontrado")
		return
	}
	if err != nil {
		h.log.Error("registration: load event", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "não foi possível enviar sua inscrição agora")
		return
	}
	if event.RegistrationMode != postgres.RegistrationForm {
		writeJSONError(w, http.StatusConflict, "este evento não recebe inscrições pelo formulário")
		return
	}

	var req registrationRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<10)).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "não foi possível ler o formulário")
		return
	}
	if strings.TrimSpace(req.Website) != "" {
		// Honeypot preenchido: robô. Responde como sucesso para não ensinar o robô a desviar.
		h.log.Warn("registration: honeypot filled", "event", event.Slug)
		writeJSON(w, http.StatusCreated, map[string]any{"status": postgres.RegistrationConfirmed})
		return
	}
	if req.ElapsedMs < minFormMs {
		writeJSONError(w, http.StatusBadRequest, "não foi possível enviar — confira os dados e tente novamente")
		return
	}
	if msg := req.validate(); msg != "" {
		writeJSONError(w, http.StatusBadRequest, msg)
		return
	}

	result, err := h.regs.Register(r.Context(), event.ID, postgres.NewRegistration{
		FullName: req.FullName, WhatsApp: req.WhatsApp, Email: req.Email,
		PracticeLevel: req.PracticeLevel, Interest: req.Interest, WeightKg: req.WeightKg,
		ConsentText: consentText(event.Name),
	})
	switch {
	case errors.Is(err, postgres.ErrRegistrationDuplicate):
		writeJSONError(w, http.StatusConflict, "este e-mail já está inscrito neste evento — confira sua caixa de entrada")
		return
	case errors.Is(err, postgres.ErrRegistrationFull):
		writeJSONError(w, http.StatusConflict, "as vagas deste evento estão esgotadas")
		return
	case errors.Is(err, postgres.ErrRegistrationClosed):
		writeJSONError(w, http.StatusConflict, "as inscrições deste evento estão encerradas")
		return
	case err != nil:
		h.log.Error("registration: save", "error", err, "event", event.Slug)
		writeJSONError(w, http.StatusInternalServerError, "não foi possível enviar sua inscrição agora")
		return
	}

	// Best-effort: a inscrição já está salva; falha de e-mail não pode desfazê-la.
	if err := h.mailer.SendEventRegistration(r.Context(), req.Email, req.FullName, mailer.RegistrationInfo{
		EventName: event.Name, DateLabel: event.DateLabel, LocationLabel: event.LocationLabel, Rules: event.RulesText,
		Waitlist: result.Status == postgres.RegistrationWaitlist, WaitlistPosition: result.WaitlistPosition,
		PagePath: "/" + event.Slug,
	}); err != nil {
		h.log.Error("registration: confirmation email", "error", err, "registrationId", result.ID)
	}

	writeJSON(w, http.StatusCreated, map[string]any{"status": result.Status, "waitlistPosition": result.WaitlistPosition})
}

// RecordView serves POST /public/events/{slug}/views — one anonymous page view.
func (h *EventRegistrationsHandler) RecordView(w http.ResponseWriter, r *http.Request) {
	event, err := h.events.GetActiveBySlug(r.Context(), chi.URLParam(r, "slug"))
	if err != nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if err := h.regs.RecordView(r.Context(), event.ID); err != nil {
		h.log.Error("record event view", "error", err)
	}
	w.WriteHeader(http.StatusNoContent)
}

// requireEvent returns the event selected in the Admin; the sign-up screens only ever
// show one event at a time.
func requireEvent(w http.ResponseWriter, r *http.Request) (string, bool) {
	id := eventScope(r)
	if id == "" {
		writeJSONError(w, http.StatusBadRequest, "selecione um evento no topo do painel")
		return "", false
	}
	return id, true
}

// AdminList serves GET /admin/registrations (inscrições + métricas do evento selecionado).
func (h *EventRegistrationsHandler) AdminList(w http.ResponseWriter, r *http.Request) {
	eventID, ok := requireEvent(w, r)
	if !ok {
		return
	}
	list, err := h.regs.List(r.Context(), eventID, r.URL.Query().Get("q"))
	if err != nil {
		h.log.Error("list registrations", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "não foi possível carregar as inscrições")
		return
	}
	stats, err := h.regs.Stats(r.Context(), eventID)
	if err != nil {
		h.log.Error("registration stats", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "não foi possível carregar as inscrições")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"registrations": list, "stats": stats})
}

var registrationStatusLabels = map[string]string{"confirmed": "Confirmada", "waitlist": "Lista de espera", "cancelled": "Cancelada"}

// AdminCSV serves GET /admin/registrations.csv.
func (h *EventRegistrationsHandler) AdminCSV(w http.ResponseWriter, r *http.Request) {
	eventID, ok := requireEvent(w, r)
	if !ok {
		return
	}
	event, err := h.events.GetByID(r.Context(), eventID)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "evento não encontrado")
		return
	}
	list, err := h.regs.List(r.Context(), eventID, "")
	if err != nil {
		h.log.Error("registrations csv", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "não foi possível gerar o arquivo")
		return
	}

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", "inscricoes-"+event.Slug+".csv"))
	_, _ = w.Write([]byte{0xEF, 0xBB, 0xBF}) // BOM: acentos certos no Excel

	cw := csv.NewWriter(w)
	cw.UseCRLF = true
	_ = cw.Write([]string{"Presença", "Nome", "WhatsApp", "E-mail", "Nível", "Interesse", "Peso (kg)", "Situação", "Inscrito em", "Check-in", "Aceite das regras", "Consentimento LGPD"})
	for _, e := range list {
		weight := ""
		if e.WeightKg != nil {
			weight = fmt.Sprint(*e.WeightKg)
		}
		checkin := "Não"
		if e.CheckedInAt != nil {
			checkin = "Sim — " + formatFortaleza(*e.CheckedInAt)
		}
		_ = cw.Write([]string{
			"☐", e.FullName, e.WhatsApp, e.Email, practiceLevels[e.PracticeLevel], interests[e.Interest], weight,
			registrationStatusLabels[e.Status], formatFortaleza(e.CreatedAt), checkin,
			formatFortaleza(e.RulesAcceptedAt), formatFortaleza(e.ConsentAt),
		})
	}
	cw.Flush()
	if err := cw.Error(); err != nil {
		h.log.Error("registrations csv write", "error", err)
	}
}

var fortaleza = func() *time.Location {
	loc, err := time.LoadLocation("America/Fortaleza")
	if err != nil {
		return time.FixedZone("BRT", -3*3600)
	}
	return loc
}()

func formatFortaleza(rfc3339 string) string {
	t, err := time.Parse(time.RFC3339, rfc3339)
	if err != nil {
		return rfc3339
	}
	return t.In(fortaleza).Format("02/01/2006 15:04")
}

// AdminSetStatus serves PATCH /admin/registrations/{id}/status.
func (h *EventRegistrationsHandler) AdminSetStatus(w http.ResponseWriter, r *http.Request) {
	eventID, ok := requireEvent(w, r)
	if !ok {
		return
	}
	var body struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || registrationStatusLabels[body.Status] == "" {
		writeJSONError(w, http.StatusBadRequest, "situação inválida")
		return
	}
	err := h.regs.SetStatus(r.Context(), eventID, chi.URLParam(r, "id"), body.Status)
	if errors.Is(err, postgres.ErrNotFound) {
		writeJSONError(w, http.StatusNotFound, "inscrição não encontrada neste evento")
		return
	}
	if err != nil {
		h.log.Error("set registration status", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "não foi possível atualizar a inscrição")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// AdminSetCheckIn serves PATCH /admin/registrations/{id}/checkin.
func (h *EventRegistrationsHandler) AdminSetCheckIn(w http.ResponseWriter, r *http.Request) {
	eventID, ok := requireEvent(w, r)
	if !ok {
		return
	}
	var body struct {
		Present bool `json:"present"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "corpo inválido")
		return
	}
	err := h.regs.SetCheckedIn(r.Context(), eventID, chi.URLParam(r, "id"), body.Present)
	if errors.Is(err, postgres.ErrNotFound) {
		writeJSONError(w, http.StatusNotFound, "inscrição não encontrada neste evento")
		return
	}
	if err != nil {
		h.log.Error("set registration check-in", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "não foi possível registrar a presença")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
