package handlers

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"

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
