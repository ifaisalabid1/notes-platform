package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/ifaisalabid1/notes-platform/internal/academic"
	"github.com/ifaisalabid1/notes-platform/internal/audit"
	"github.com/ifaisalabid1/notes-platform/internal/views"
)

type AdminSemesterHandler struct {
	classRepo      *academic.ClassRepository
	semesterRepo   *academic.SemesterRepository
	sessionManager *scs.SessionManager
	auditRepo      *audit.Repository
	renderer       *views.Renderer
}

func NewAdminSemesterHandler(
	classRepo *academic.ClassRepository,
	semesterRepo *academic.SemesterRepository,
	sessionManager *scs.SessionManager,
	auditRepo *audit.Repository,
	renderer *views.Renderer,
) *AdminSemesterHandler {
	return &AdminSemesterHandler{
		classRepo:      classRepo,
		semesterRepo:   semesterRepo,
		sessionManager: sessionManager,
		auditRepo:      auditRepo,
		renderer:       renderer,
	}
}

type AdminSemestersPageData struct {
	Classes   []academic.Class
	Semesters []academic.Semester
}

type AdminSemesterEditPageData struct {
	Classes  []academic.Class
	Semester academic.Semester
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

	createdSemester, err := h.semesterRepo.Create(r.Context(), academic.CreateSemesterParams{
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

	writeAuditLog(
		r,
		h.sessionManager,
		h.auditRepo,
		"semester_created",
		"semester",
		&createdSemester.ID,
		"Created semester",
		map[string]any{
			"name":         createdSemester.Name,
			"slug":         createdSemester.Slug,
			"class_id":     createdSemester.ClassID.String(),
			"is_published": createdSemester.IsPublished,
			"sort_order":   createdSemester.SortOrder,
		},
	)

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

func (h *AdminSemesterHandler) Edit(w http.ResponseWriter, r *http.Request) {
	semesterID, err := uuid.Parse(chi.URLParam(r, "semesterID"))
	if err != nil {
		http.NotFound(w, r)
		return
	}

	pageData, err := h.editPageData(r, semesterID)
	if err != nil {
		if errors.Is(err, academic.ErrSemesterNotFound) {
			http.NotFound(w, r)
			return
		}

		slog.Error("failed to load semester edit page", "error", err)
		http.Error(w, "Failed to load semester", http.StatusInternalServerError)
		return
	}

	h.renderer.Render(w, r, "admin_semester_edit.tmpl", views.TemplateData{
		Title: "Edit Semester",
		Data:  pageData,
	})
}

func (h *AdminSemesterHandler) Update(w http.ResponseWriter, r *http.Request) {
	semesterID, err := uuid.Parse(chi.URLParam(r, "semesterID"))
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if err := r.ParseForm(); err != nil {
		h.renderEditWithError(w, r, semesterID, "Invalid form submission.")
		return
	}

	classIDValue := strings.TrimSpace(r.PostForm.Get("class_id"))
	name := strings.TrimSpace(r.PostForm.Get("name"))
	description := strings.TrimSpace(r.PostForm.Get("description"))
	sortOrderValue := strings.TrimSpace(r.PostForm.Get("sort_order"))
	isPublished := r.PostForm.Get("is_published") == "on"

	classID, err := uuid.Parse(classIDValue)
	if err != nil {
		h.renderEditWithError(w, r, semesterID, "Please select a valid class.")
		return
	}

	sortOrder := 0
	if sortOrderValue != "" {
		parsedSortOrder, err := strconv.Atoi(sortOrderValue)
		if err != nil {
			h.renderEditWithError(w, r, semesterID, "Sort order must be a number.")
			return
		}

		sortOrder = parsedSortOrder
	}

	if name == "" {
		h.renderEditWithError(w, r, semesterID, "Semester name is required.")
		return
	}

	updatedSemester, err := h.semesterRepo.Update(r.Context(), academic.UpdateSemesterParams{
		ID:          semesterID,
		ClassID:     classID,
		Name:        name,
		Description: description,
		SortOrder:   sortOrder,
		IsPublished: isPublished,
	})
	if err != nil {
		if isUniqueViolation(err) {
			h.renderEditWithError(w, r, semesterID, "This semester already exists for the selected class.")
			return
		}

		if errors.Is(err, academic.ErrSemesterNotFound) {
			http.NotFound(w, r)
			return
		}

		slog.Error("failed to update semester", "error", err)
		h.renderEditWithError(w, r, semesterID, "Failed to update semester.")
		return
	}

	writeAuditLog(
		r,
		h.sessionManager,
		h.auditRepo,
		"semester_updated",
		"semester",
		&updatedSemester.ID,
		"Updated semester",
		map[string]any{
			"name":         updatedSemester.Name,
			"slug":         updatedSemester.Slug,
			"class_id":     updatedSemester.ClassID.String(),
			"is_published": updatedSemester.IsPublished,
			"sort_order":   updatedSemester.SortOrder,
		},
	)

	http.Redirect(w, r, "/admin/semesters", http.StatusSeeOther)
}

func (h *AdminSemesterHandler) renderEditWithError(w http.ResponseWriter, r *http.Request, semesterID uuid.UUID, message string) {
	pageData, err := h.editPageData(r, semesterID)
	if err != nil {
		slog.Error("failed to reload semester edit page after error", "error", err)
		http.Error(w, "Failed to load semester", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusUnprocessableEntity)

	h.renderer.Render(w, r, "admin_semester_edit.tmpl", views.TemplateData{
		Title: "Edit Semester",
		Error: message,
		Data:  pageData,
	})
}

func (h *AdminSemesterHandler) editPageData(r *http.Request, semesterID uuid.UUID) (AdminSemesterEditPageData, error) {
	classes, err := h.classRepo.List(r.Context())
	if err != nil {
		return AdminSemesterEditPageData{}, err
	}

	semesterItem, err := h.semesterRepo.FindByID(r.Context(), semesterID)
	if err != nil {
		return AdminSemesterEditPageData{}, err
	}

	return AdminSemesterEditPageData{
		Classes:  classes,
		Semester: semesterItem,
	}, nil
}
