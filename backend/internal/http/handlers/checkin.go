package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"

	"p5wellness/backend/internal/domain/auth"
	"p5wellness/backend/internal/http/middleware"
	"p5wellness/backend/internal/qrcode"
	"p5wellness/backend/internal/repository/postgres"
)

type CheckinHandler struct {
	checkin  *postgres.CheckinRepository
	qrSecret string
	log      *slog.Logger
}

func NewCheckinHandler(checkin *postgres.CheckinRepository, qrSecret string, log *slog.Logger) *CheckinHandler {
	return &CheckinHandler{checkin: checkin, qrSecret: qrSecret, log: log}
}

type validateRequest struct {
	Token           string `json:"token"`
	ClientScanID    string `json:"clientScanId"`
	DeviceScannedAt string `json:"deviceScannedAt"`
	// EventID is the event the terminal was set to when the scan was taken — offline scans
	// are replayed later, possibly after the terminal switched events. "" = the request's
	// own event header.
	EventID string `json:"eventId"`
}

type validateResponse struct {
	Result      string `json:"result"`
	Label       string `json:"label,omitempty"`
	VendorName  string `json:"vendorName,omitempty"`
	StudentName string `json:"studentName,omitempty"`
	OrderNumber string `json:"orderNumber,omitempty"`
	UsedAt      string `json:"usedAt,omitempty"`
	Message     string `json:"message"`
}

func (h *CheckinHandler) Validate(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())

	var req validateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Token == "" {
		writeJSONError(w, http.StatusBadRequest, "QR Code inválido")
		return
	}

	writeJSON(w, http.StatusOK, h.validateOne(r.Context(), claims, scanEvent(r, req.EventID), req))
}

// scanEvent is the event a scan is checked against: the one recorded with the scan, or
// the terminal's current event header.
func scanEvent(r *http.Request, recorded string) string {
	if _, err := uuid.Parse(recorded); err == nil {
		return recorded
	}
	return eventScope(r)
}

type batchValidateRequest struct {
	Scans []validateRequest `json:"scans"`
}

type batchValidateResult struct {
	ClientScanID string `json:"clientScanId"`
	validateResponse
}

// ValidateBatch replays a queue of scans made while the check-in device was offline.
// Each scan carries its own clientScanId — already used by ValidateToken's
// ON CONFLICT DO NOTHING logging — so replaying the same batch twice (e.g. a retry
// after a dropped connection) is a safe no-op, not a double check-in.
func (h *CheckinHandler) ValidateBatch(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())

	var req batchValidateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "corpo inválido")
		return
	}
	if len(req.Scans) > 200 {
		writeJSONError(w, http.StatusBadRequest, "lote grande demais — envie no máximo 200 por vez")
		return
	}

	results := make([]batchValidateResult, 0, len(req.Scans))
	for _, scan := range req.Scans {
		if scan.Token == "" {
			continue
		}
		results = append(results, batchValidateResult{
			ClientScanID:     scan.ClientScanID,
			validateResponse: h.validateOne(r.Context(), claims, scanEvent(r, scan.EventID), scan),
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{"results": results})
}

func (h *CheckinHandler) validateOne(ctx context.Context, claims *auth.Claims, eventID string, req validateRequest) validateResponse {
	if _, ok := qrcode.Verify(req.Token, h.qrSecret); !ok {
		return validateResponse{Result: "invalid_signature", Message: "Código não reconhecido."}
	}

	clientScanID, scannedAt := scanMeta(req.ClientScanID, req.DeviceScannedAt)

	ticket, err := h.checkin.ValidateTokenForEvent(ctx, req.Token, eventID, claims.VendorID, claims.UserID(), clientScanID, scannedAt)
	if err != nil {
		h.log.Error("validate token", "error", err)
		return validateResponse{Result: "error", Message: "não foi possível validar agora"}
	}
	return ticketResponse(ticket)
}

func scanMeta(clientScanID, deviceScannedAt string) (string, time.Time) {
	if clientScanID == "" {
		clientScanID = uuid.NewString()
	}
	scannedAt := time.Now()
	if deviceScannedAt != "" {
		if parsed, err := time.Parse(time.RFC3339, deviceScannedAt); err == nil {
			scannedAt = parsed
		}
	}
	return clientScanID, scannedAt
}

func ticketResponse(ticket *postgres.ValidatedTicket) validateResponse {
	var usedAt string
	if ticket.UsedAt != nil {
		usedAt = ticket.UsedAt.Format(time.RFC3339)
	}
	message := messageFor(ticket.Result)
	if ticket.Result == postgres.ResultWrongEvent && ticket.EventName != "" {
		message = "Este ingresso é do evento \"" + ticket.EventName + "\" — não pode ser validado neste evento."
	}
	return validateResponse{
		Result: string(ticket.Result), Label: ticket.Label, VendorName: ticket.VendorName,
		StudentName: ticket.StudentName, OrderNumber: ticket.OrderNumber, UsedAt: usedAt,
		Message: message,
	}
}

type validateByIDRequest struct {
	EntitlementID   string `json:"entitlementId"`
	ClientScanID    string `json:"clientScanId"`
	DeviceScannedAt string `json:"deviceScannedAt"`
}

// ValidateByID is the second check-in path: staff pick the guest off the paid roster
// (Roster below) and confirm by id instead of scanning a QR code — useful when a guest's
// phone is dead, the QR won't scan, or it's just faster to tap a name off a short list.
func (h *CheckinHandler) ValidateByID(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())

	var req validateByIDRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.EntitlementID == "" {
		writeJSONError(w, http.StatusBadRequest, "ficha inválida")
		return
	}

	clientScanID, scannedAt := scanMeta(req.ClientScanID, req.DeviceScannedAt)

	ticket, err := h.checkin.ValidateByIDForEvent(r.Context(), req.EntitlementID, eventScope(r), claims.VendorID, claims.UserID(), clientScanID, scannedAt)
	if err != nil {
		h.log.Error("validate by id", "error", err)
		writeJSON(w, http.StatusOK, validateResponse{Result: "error", Message: "não foi possível validar agora"})
		return
	}
	writeJSON(w, http.StatusOK, ticketResponse(ticket))
}

// Roster lists everyone who paid for a class or breakfast on a given day (default:
// today), grouped by turma / Café da Manhã, for the list-based check-in screen.
// scannerVendorID (from the caller's own login) keeps a staff account seeing only its
// own sector's guests, exactly like scanning does.
func (h *CheckinHandler) Roster(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())

	date := r.URL.Query().Get("date")
	if date == "" {
		date = postgres.TodayInEventTZ()
	}

	groups, err := h.checkin.ListRoster(r.Context(), claims.VendorID, date, eventScope(r))
	if err != nil {
		h.log.Error("list roster", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "não foi possível carregar a lista")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"date": date, "groups": groups})
}

// RosterDates lists the event dates the check-in list's date picker offers, pulled
// straight from registered turmas instead of assuming "today" is the only valid day.
func (h *CheckinHandler) RosterDates(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())

	dates, err := h.checkin.ListRosterDates(r.Context(), claims.VendorID, eventScope(r))
	if err != nil {
		h.log.Error("list roster dates", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "não foi possível carregar as datas")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"dates": dates})
}

func messageFor(r postgres.ValidationResult) string {
	switch r {
	case postgres.ResultSuccess:
		return "Check-in confirmado."
	case postgres.ResultAlreadyUsed:
		return "Este benefício já foi utilizado."
	case postgres.ResultExpired:
		return "Este benefício expirou."
	case postgres.ResultNotFound:
		return "QR Code não encontrado."
	case postgres.ResultWrongSector:
		return "Este QR Code pertence a outro setor — não pode ser validado por aqui."
	case postgres.ResultWrongEvent:
		return "Este ingresso é de outro evento — não pode ser validado neste evento."
	default:
		return "Não foi possível validar."
	}
}
