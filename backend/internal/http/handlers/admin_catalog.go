package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"p5wellness/backend/internal/repository/postgres"
)

type AdminCatalogHandler struct {
	repo    *postgres.AdminCatalogRepository
	vendors *postgres.VendorRepository
	log     *slog.Logger
}

func NewAdminCatalogHandler(repo *postgres.AdminCatalogRepository, vendors *postgres.VendorRepository, log *slog.Logger) *AdminCatalogHandler {
	return &AdminCatalogHandler{repo: repo, vendors: vendors, log: log}
}

func (h *AdminCatalogHandler) ListVendors(w http.ResponseWriter, r *http.Request) {
	vendors, err := h.vendors.List(r.Context())
	if err != nil {
		h.log.Error("list vendors", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "não foi possível carregar os setores")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"vendors": vendors})
}

// ── Activities ──────────────────────────────────────────────────────────────

func (h *AdminCatalogHandler) ListActivities(w http.ResponseWriter, r *http.Request) {
	activities, err := h.repo.ListActivities(r.Context())
	if err != nil {
		h.log.Error("list activities", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "não foi possível carregar as atividades")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"activities": activities})
}

type activityRequest struct {
	Title           string `json:"title"`
	Instructor      string `json:"instructor"`
	DurationMinutes int    `json:"durationMinutes"`
	Description     string `json:"description"`
	VendorID        string `json:"vendorId"`
}

func (h *AdminCatalogHandler) CreateActivity(w http.ResponseWriter, r *http.Request) {
	var req activityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Title == "" {
		writeJSONError(w, http.StatusBadRequest, "título é obrigatório")
		return
	}
	if req.VendorID == "" {
		writeJSONError(w, http.StatusBadRequest, "selecione o setor responsável pela atividade")
		return
	}
	id, err := h.repo.CreateActivity(r.Context(), req.Title, req.Instructor, req.DurationMinutes, req.Description, req.VendorID)
	if err != nil {
		h.log.Error("create activity", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "não foi possível criar a atividade")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"id": id})
}

func (h *AdminCatalogHandler) UpdateActivity(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req activityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Title == "" {
		writeJSONError(w, http.StatusBadRequest, "título é obrigatório")
		return
	}
	if req.VendorID == "" {
		writeJSONError(w, http.StatusBadRequest, "selecione o setor responsável pela atividade")
		return
	}
	if err := h.repo.UpdateActivity(r.Context(), id, req.Title, req.Instructor, req.DurationMinutes, req.Description, req.VendorID); err != nil {
		h.log.Error("update activity", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "não foi possível salvar a atividade")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *AdminCatalogHandler) SetActivityActive(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		Active bool `json:"active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "corpo inválido")
		return
	}
	if err := h.repo.SetActivityActive(r.Context(), id, body.Active); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "não foi possível atualizar")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ── Products ────────────────────────────────────────────────────────────────

func (h *AdminCatalogHandler) ListProducts(w http.ResponseWriter, r *http.Request) {
	products, err := h.repo.ListProducts(r.Context())
	if err != nil {
		h.log.Error("list products", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "não foi possível carregar os produtos")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"products": products})
}

type productRequest struct {
	Title             string   `json:"title"`
	Description       string   `json:"description"`
	Type              string   `json:"type"`
	IncludesBreakfast bool     `json:"includesBreakfast"`
	PriceCents        int      `json:"priceCents"`
	Featured          bool     `json:"featured"`
	ChooseOneActivity bool     `json:"chooseOneActivity"`
	ActivityIDs       []string `json:"activityIds"`
}

func (h *AdminCatalogHandler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	var req productRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Title == "" || req.PriceCents <= 0 {
		writeJSONError(w, http.StatusBadRequest, "título e preço válido são obrigatórios")
		return
	}
	if req.Type != "class" && req.Type != "combo" {
		writeJSONError(w, http.StatusBadRequest, "tipo deve ser class ou combo")
		return
	}
	if len(req.ActivityIDs) == 0 && !req.IncludesBreakfast {
		writeJSONError(w, http.StatusBadRequest, "selecione ao menos uma atividade ou inclua o café da manhã")
		return
	}
	if req.ChooseOneActivity && len(req.ActivityIDs) < 2 {
		writeJSONError(w, http.StatusBadRequest, "\"escolher uma atividade\" exige pelo menos 2 atividades vinculadas")
		return
	}

	id, err := h.repo.CreateProduct(r.Context(), postgres.CreateProductParams{
		Title: req.Title, Description: req.Description, Type: req.Type,
		IncludesBreakfast: req.IncludesBreakfast, PriceCents: req.PriceCents,
		Featured: req.Featured, ChooseOneActivity: req.ChooseOneActivity, ActivityIDs: req.ActivityIDs,
	})
	if err != nil {
		h.log.Error("create product", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "não foi possível publicar o produto")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"id": id})
}

func (h *AdminCatalogHandler) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req productRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Title == "" || req.PriceCents <= 0 {
		writeJSONError(w, http.StatusBadRequest, "título e preço válido são obrigatórios")
		return
	}
	if req.Type != "class" && req.Type != "combo" {
		writeJSONError(w, http.StatusBadRequest, "tipo deve ser class ou combo")
		return
	}
	if len(req.ActivityIDs) == 0 && !req.IncludesBreakfast {
		writeJSONError(w, http.StatusBadRequest, "selecione ao menos uma atividade ou inclua o café da manhã")
		return
	}
	if req.ChooseOneActivity && len(req.ActivityIDs) < 2 {
		writeJSONError(w, http.StatusBadRequest, "\"escolher uma atividade\" exige pelo menos 2 atividades vinculadas")
		return
	}
	err := h.repo.UpdateProduct(r.Context(), id, postgres.CreateProductParams{
		Title: req.Title, Description: req.Description, Type: req.Type,
		IncludesBreakfast: req.IncludesBreakfast, PriceCents: req.PriceCents,
		Featured: req.Featured, ChooseOneActivity: req.ChooseOneActivity, ActivityIDs: req.ActivityIDs,
	})
	if err != nil {
		h.log.Error("update product", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "não foi possível salvar o produto")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *AdminCatalogHandler) SetProductActive(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		Active bool `json:"active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "corpo inválido")
		return
	}
	if err := h.repo.SetProductActive(r.Context(), id, body.Active); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "não foi possível atualizar")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
