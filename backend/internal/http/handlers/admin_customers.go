package handlers

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"p5wellness/backend/internal/repository/postgres"
)

type AdminCustomersHandler struct {
	repo     *postgres.AdminCustomerRepository
	students *postgres.StudentRepository
	log      *slog.Logger
}

func NewAdminCustomersHandler(repo *postgres.AdminCustomerRepository, students *postgres.StudentRepository, log *slog.Logger) *AdminCustomersHandler {
	return &AdminCustomersHandler{repo: repo, students: students, log: log}
}

func (h *AdminCustomersHandler) Search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if len(query) == 1 {
		writeJSONError(w, http.StatusBadRequest, "digite ao menos 2 caracteres para buscar")
		return
	}
	customers, err := h.repo.Search(r.Context(), query)
	if err != nil {
		h.log.Error("search customers", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "não foi possível buscar clientes")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"customers": customers})
}

func (h *AdminCustomersHandler) Detail(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	detail, err := h.repo.Detail(r.Context(), id)
	if err != nil {
		if err == postgres.ErrNotFound {
			writeJSONError(w, http.StatusNotFound, "cliente não encontrado")
			return
		}
		h.log.Error("customer detail", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "não foi possível carregar o cliente")
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

// Anonymize fulfils an LGPD art. 18 deletion request: scrubs the student's personal
// data in place (name, e-mail, phone, CPF) while keeping the order/payment rows for
// fiscal retention. Irreversible — the account can no longer log in afterwards.
func (h *AdminCustomersHandler) Anonymize(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.students.Anonymize(r.Context(), id); err != nil {
		h.log.Error("anonymize student", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "não foi possível anonimizar o cadastro")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
