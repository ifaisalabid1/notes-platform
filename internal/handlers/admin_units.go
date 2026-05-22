package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/ifaisalabid1/notes-platform/internal/academic"
	"github.com/ifaisalabid1/notes-platform/internal/views"
)

type AdminUnitHandler struct {
	subjectRepo *academic.SubjectRepository
	unitRepo    *academic.UnitRepository
	renderer    *views.Renderer
}

func NewAdminUnitHandler(
	subjectRepo *academic.SubjectRepository,
	unitRepo *academic.UnitRepository,
	renderer *views.Renderer,
) *AdminUnitHandler {
	return &AdminUnitHandler{
		subjectRepo: subjectRepo,
		unitRepo:    unitRepo,
		renderer:    renderer,
	}
}

type AdminUnitsPageData struct {
	Subjects []academic.Subject
	Units    []academic.Unit
}

type AdminUnitEditPageData struct {
	Subjects []academic.Subject
	Unit     academic.Unit
}

func (h *AdminUnitHandler) Index(w http.ResponseWriter, r *http.Request) {
	pageData, err := h.pageData(r)
	if err != nil {
		slog.Error("failed to load units page data", "error", err)
		http.Error(w, "Failed to load units", http.StatusInternalServerError)
		return
	}

	h.renderer.Render(w, r, "admin_units.tmpl", views.TemplateData{
		Title: "Units",
		Data:  pageData,
	})
}

func (h *AdminUnitHandler) Store(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.renderIndexWithError(w, r, "Invalid form submission.")
		return
	}

	subjectIDValue := strings.TrimSpace(r.PostForm.Get("subject_id"))
	name := strings.TrimSpace(r.PostForm.Get("name"))
	description := strings.TrimSpace(r.PostForm.Get("description"))
	sortOrderValue := strings.TrimSpace(r.PostForm.Get("sort_order"))
	isPublished := r.PostForm.Get("is_published") == "on"

	subjectID, err := uuid.Parse(subjectIDValue)
	if err != nil {
		h.renderIndexWithError(w, r, "Please select a valid subject.")
		return
	}

	sortOrder := 0
	if sortOrderValue != "" {
		parsedSortOrder, err := strconv.Atoi(sortOrderValue)
		if err != nil {
			h.renderIndexWithError(w, r, "Sort order must be a number.")
			return
		}

		sortOrder = parsedSortOrder
	}

	if name == "" {
		h.renderIndexWithError(w, r, "Unit name is required.")
		return
	}

	_, err = h.unitRepo.Create(r.Context(), academic.CreateUnitParams{
		SubjectID:   subjectID,
		Name:        name,
		Description: description,
		SortOrder:   sortOrder,
		IsPublished: isPublished,
	})
	if err != nil {
		if isUniqueViolation(err) {
			h.renderIndexWithError(w, r, "This unit already exists for the selected subject.")
			return
		}

		slog.Error("failed to create unit", "error", err)
		h.renderIndexWithError(w, r, "Failed to create unit.")
		return
	}

	http.Redirect(w, r, "/admin/units", http.StatusSeeOther)
}

func (h *AdminUnitHandler) renderIndexWithError(w http.ResponseWriter, r *http.Request, message string) {
	pageData, err := h.pageData(r)
	if err != nil {
		slog.Error("failed to load units page data after error", "error", err)
		http.Error(w, "Failed to load units", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusUnprocessableEntity)

	h.renderer.Render(w, r, "admin_units.tmpl", views.TemplateData{
		Title: "Units",
		Error: message,
		Data:  pageData,
	})
}

func (h *AdminUnitHandler) pageData(r *http.Request) (AdminUnitsPageData, error) {
	subjects, err := h.subjectRepo.List(r.Context())
	if err != nil {
		return AdminUnitsPageData{}, err
	}

	units, err := h.unitRepo.List(r.Context())
	if err != nil {
		return AdminUnitsPageData{}, err
	}

	return AdminUnitsPageData{
		Subjects: subjects,
		Units:    units,
	}, nil
}

func (h *AdminUnitHandler) Edit(w http.ResponseWriter, r *http.Request) {
	unitID, err := uuid.Parse(chi.URLParam(r, "unitID"))
	if err != nil {
		http.NotFound(w, r)
		return
	}

	pageData, err := h.editPageData(r, unitID)
	if err != nil {
		if errors.Is(err, academic.ErrUnitNotFound) {
			http.NotFound(w, r)
			return
		}

		slog.Error("failed to load unit edit page", "error", err)
		http.Error(w, "Failed to load unit", http.StatusInternalServerError)
		return
	}

	h.renderer.Render(w, r, "admin_unit_edit.tmpl", views.TemplateData{
		Title: "Edit Unit",
		Data:  pageData,
	})
}

func (h *AdminUnitHandler) Update(w http.ResponseWriter, r *http.Request) {
	unitID, err := uuid.Parse(chi.URLParam(r, "unitID"))
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if err := r.ParseForm(); err != nil {
		h.renderEditWithError(w, r, unitID, "Invalid form submission.")
		return
	}

	subjectIDValue := strings.TrimSpace(r.PostForm.Get("subject_id"))
	name := strings.TrimSpace(r.PostForm.Get("name"))
	description := strings.TrimSpace(r.PostForm.Get("description"))
	sortOrderValue := strings.TrimSpace(r.PostForm.Get("sort_order"))
	isPublished := r.PostForm.Get("is_published") == "on"

	subjectID, err := uuid.Parse(subjectIDValue)
	if err != nil {
		h.renderEditWithError(w, r, unitID, "Please select a valid subject.")
		return
	}

	sortOrder := 0
	if sortOrderValue != "" {
		parsedSortOrder, err := strconv.Atoi(sortOrderValue)
		if err != nil {
			h.renderEditWithError(w, r, unitID, "Sort order must be a number.")
			return
		}

		sortOrder = parsedSortOrder
	}

	if name == "" {
		h.renderEditWithError(w, r, unitID, "Unit name is required.")
		return
	}

	_, err = h.unitRepo.Update(r.Context(), academic.UpdateUnitParams{
		ID:          unitID,
		SubjectID:   subjectID,
		Name:        name,
		Description: description,
		SortOrder:   sortOrder,
		IsPublished: isPublished,
	})
	if err != nil {
		if isUniqueViolation(err) {
			h.renderEditWithError(w, r, unitID, "This unit already exists for the selected subject.")
			return
		}

		if errors.Is(err, academic.ErrUnitNotFound) {
			http.NotFound(w, r)
			return
		}

		slog.Error("failed to update unit", "error", err)
		h.renderEditWithError(w, r, unitID, "Failed to update unit.")
		return
	}

	http.Redirect(w, r, "/admin/units", http.StatusSeeOther)
}

func (h *AdminUnitHandler) renderEditWithError(w http.ResponseWriter, r *http.Request, unitID uuid.UUID, message string) {
	pageData, err := h.editPageData(r, unitID)
	if err != nil {
		slog.Error("failed to reload unit edit page after error", "error", err)
		http.Error(w, "Failed to load unit", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusUnprocessableEntity)

	h.renderer.Render(w, r, "admin_unit_edit.tmpl", views.TemplateData{
		Title: "Edit Unit",
		Error: message,
		Data:  pageData,
	})
}

func (h *AdminUnitHandler) editPageData(r *http.Request, unitID uuid.UUID) (AdminUnitEditPageData, error) {
	subjects, err := h.subjectRepo.List(r.Context())
	if err != nil {
		return AdminUnitEditPageData{}, err
	}

	unitItem, err := h.unitRepo.FindByID(r.Context(), unitID)
	if err != nil {
		return AdminUnitEditPageData{}, err
	}

	return AdminUnitEditPageData{
		Subjects: subjects,
		Unit:     unitItem,
	}, nil
}
