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

type AdminClassHandler struct {
	classRepo      *academic.ClassRepository
	sessionManager *scs.SessionManager
	auditRepo      *audit.Repository
	renderer       *views.Renderer
}

func NewAdminClassHandler(
	classRepo *academic.ClassRepository,
	sessionManager *scs.SessionManager,
	auditRepo *audit.Repository,
	renderer *views.Renderer,
) *AdminClassHandler {
	return &AdminClassHandler{
		classRepo:      classRepo,
		sessionManager: sessionManager,
		auditRepo:      auditRepo,
		renderer:       renderer,
	}
}

type AdminClassesPageData struct {
	Classes []academic.Class
}

type AdminClassEditPageData struct {
	Class academic.Class
}

func (h *AdminClassHandler) Index(w http.ResponseWriter, r *http.Request) {
	classes, err := h.classRepo.List(r.Context())
	if err != nil {
		slog.Error("failed to list classes", "error", err)
		http.Error(w, "Failed to load classes", http.StatusInternalServerError)
		return
	}

	h.renderer.Render(w, r, "admin_classes.tmpl", views.TemplateData{
		Title: "Classes",
		Data: AdminClassesPageData{
			Classes: classes,
		},
	})
}

func (h *AdminClassHandler) Store(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.renderIndexWithError(w, r, "Invalid form submission.")
		return
	}

	name := strings.TrimSpace(r.PostForm.Get("name"))
	description := strings.TrimSpace(r.PostForm.Get("description"))
	sortOrderValue := strings.TrimSpace(r.PostForm.Get("sort_order"))
	isPublished := r.PostForm.Get("is_published") == "on"

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
		h.renderIndexWithError(w, r, "Class name is required.")
		return
	}

	createdClass, err := h.classRepo.Create(r.Context(), academic.CreateClassParams{
		Name:        name,
		Description: description,
		SortOrder:   sortOrder,
		IsPublished: isPublished,
	})
	if err != nil {
		if isUniqueViolation(err) {
			h.renderIndexWithError(w, r, "A class with this name already exists.")
			return
		}

		slog.Error("failed to create class", "error", err)
		h.renderIndexWithError(w, r, "Failed to create class.")
		return
	}

	writeAuditLog(
		r,
		h.sessionManager,
		h.auditRepo,
		"class_created",
		"class",
		&createdClass.ID,
		"Created class",
		map[string]any{
			"name":         createdClass.Name,
			"slug":         createdClass.Slug,
			"is_published": createdClass.IsPublished,
			"sort_order":   createdClass.SortOrder,
		},
	)

	http.Redirect(w, r, "/admin/classes", http.StatusSeeOther)
}

func (h *AdminClassHandler) renderIndexWithError(w http.ResponseWriter, r *http.Request, message string) {
	classes, err := h.classRepo.List(r.Context())
	if err != nil {
		slog.Error("failed to list classes after form error", "error", err)
		http.Error(w, "Failed to load classes", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusUnprocessableEntity)

	h.renderer.Render(w, r, "admin_classes.tmpl", views.TemplateData{
		Title: "Classes",
		Error: message,
		Data: AdminClassesPageData{
			Classes: classes,
		},
	})
}

func (h *AdminClassHandler) Edit(w http.ResponseWriter, r *http.Request) {
	classID, err := uuid.Parse(chi.URLParam(r, "classID"))
	if err != nil {
		http.NotFound(w, r)
		return
	}

	classItem, err := h.classRepo.FindByID(r.Context(), classID)
	if err != nil {
		if errors.Is(err, academic.ErrClassNotFound) {
			http.NotFound(w, r)
			return
		}

		slog.Error("failed to find class for edit", "error", err)
		http.Error(w, "Failed to load class", http.StatusInternalServerError)
		return
	}

	h.renderer.Render(w, r, "admin_class_edit.tmpl", views.TemplateData{
		Title: "Edit Class",
		Data: AdminClassEditPageData{
			Class: classItem,
		},
	})
}

func (h *AdminClassHandler) Update(w http.ResponseWriter, r *http.Request) {
	classID, err := uuid.Parse(chi.URLParam(r, "classID"))
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if err := r.ParseForm(); err != nil {
		h.renderEditWithError(w, r, classID, "Invalid form submission.")
		return
	}

	name := strings.TrimSpace(r.PostForm.Get("name"))
	description := strings.TrimSpace(r.PostForm.Get("description"))
	sortOrderValue := strings.TrimSpace(r.PostForm.Get("sort_order"))
	isPublished := r.PostForm.Get("is_published") == "on"

	sortOrder := 0
	if sortOrderValue != "" {
		parsedSortOrder, err := strconv.Atoi(sortOrderValue)
		if err != nil {
			h.renderEditWithError(w, r, classID, "Sort order must be a number.")
			return
		}

		sortOrder = parsedSortOrder
	}

	if name == "" {
		h.renderEditWithError(w, r, classID, "Class name is required.")
		return
	}

	updatedClass, err := h.classRepo.Update(r.Context(), academic.UpdateClassParams{
		ID:          classID,
		Name:        name,
		Description: description,
		SortOrder:   sortOrder,
		IsPublished: isPublished,
	})
	if err != nil {
		if isUniqueViolation(err) {
			h.renderEditWithError(w, r, classID, "A class with this name already exists.")
			return
		}

		if errors.Is(err, academic.ErrClassNotFound) {
			http.NotFound(w, r)
			return
		}

		slog.Error("failed to update class", "error", err)
		h.renderEditWithError(w, r, classID, "Failed to update class.")
		return
	}

	writeAuditLog(
		r,
		h.sessionManager,
		h.auditRepo,
		"class_updated",
		"class",
		&updatedClass.ID,
		"Updated class",
		map[string]any{
			"name":         updatedClass.Name,
			"slug":         updatedClass.Slug,
			"is_published": updatedClass.IsPublished,
			"sort_order":   updatedClass.SortOrder,
		},
	)

	http.Redirect(w, r, "/admin/classes", http.StatusSeeOther)
}

func (h *AdminClassHandler) renderEditWithError(w http.ResponseWriter, r *http.Request, classID uuid.UUID, message string) {
	classItem, err := h.classRepo.FindByID(r.Context(), classID)
	if err != nil {
		slog.Error("failed to reload class edit page after error", "error", err)
		http.Error(w, "Failed to load class", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusUnprocessableEntity)

	h.renderer.Render(w, r, "admin_class_edit.tmpl", views.TemplateData{
		Title: "Edit Class",
		Error: message,
		Data: AdminClassEditPageData{
			Class: classItem,
		},
	})
}
