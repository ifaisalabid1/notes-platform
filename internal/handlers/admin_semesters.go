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

type AdminSemesterHandler struct {
	classRepo    *academic.ClassRepository
	semesterRepo *academic.SemesterRepository
	renderer     *views.Renderer
}

func NewAdminSemesterHandler(
	classRepo *academic.ClassRepository,
	semesterRepo *academic.SemesterRepository,
	renderer *views.Renderer,
) *AdminSemesterHandler {
	return &AdminSemesterHandler{
		classRepo:    classRepo,
		semesterRepo: semesterRepo,
		renderer:     renderer,
	}
}

type AdminSemestersPageData struct {
	Classes   []academic.Class
	Semesters []academic.Semester
}

func (h *AdminSemesterHandler) Index(w http.ResponseWriter, r *http.Request) {
	pageData, err := h.pageData(r)
	if err != nil {
		slog.Error("failed to load semesters page data", "error", err)
		http.Error(w, "Failed to load semesters", http.StatusInternalServerError)
		return
	}

	h.renderer.Render(w, r, "admin_semesters.tmpl", views.TemplateData{
		Title: "Semesters",
		Data:  pageData,
	})
}

func (h *AdminSemesterHandler) Store(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.renderIndexWithError(w, r, "Invalid form submission.")
		return
	}

	classIDValue := strings.TrimSpace(r.PostForm.Get("class_id"))
	name := strings.TrimSpace(r.PostForm.Get("name"))
	description := strings.TrimSpace(r.PostForm.Get("description"))
	sortOrderValue := strings.TrimSpace(r.PostForm.Get("sort_order"))
	isPublished := r.PostForm.Get("is_published") == "on"

	classID, err := uuid.Parse(classIDValue)
	if err != nil {
		h.renderIndexWithError(w, r, "Please select a valid class.")
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
		h.renderIndexWithError(w, r, "Semester name is required.")
		return
	}

	_, err = h.semesterRepo.Create(r.Context(), academic.CreateSemesterParams{
		ClassID:     classID,
		Name:        name,
		Description: description,
		SortOrder:   sortOrder,
		IsPublished: isPublished,
	})
	if err != nil {
		if isUniqueViolation(err) {
			h.renderIndexWithError(w, r, "This semester already exists for the selected class.")
			return
		}

		slog.Error("failed to create semester", "error", err)
		h.renderIndexWithError(w, r, "Failed to create semester.")
		return
	}

	http.Redirect(w, r, "/admin/semesters", http.StatusSeeOther)
}

func (h *AdminSemesterHandler) renderIndexWithError(w http.ResponseWriter, r *http.Request, message string) {
	pageData, err := h.pageData(r)
	if err != nil {
		slog.Error("failed to load semesters page data after error", "error", err)
		http.Error(w, "Failed to load semesters", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusUnprocessableEntity)

	h.renderer.Render(w, r, "admin_semesters.tmpl", views.TemplateData{
		Title: "Semesters",
		Error: message,
		Data:  pageData,
	})
}

func (h *AdminSemesterHandler) pageData(r *http.Request) (AdminSemestersPageData, error) {
	classes, err := h.classRepo.List(r.Context())
	if err != nil {
		return AdminSemestersPageData{}, err
	}

	semesters, err := h.semesterRepo.List(r.Context())
	if err != nil {
		return AdminSemestersPageData{}, err
	}

	return AdminSemestersPageData{
		Classes:   classes,
		Semesters: semesters,
	}, nil
}
