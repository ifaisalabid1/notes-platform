package handlers

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/ifaisalabid1/notes-platform/internal/academic"
	"github.com/ifaisalabid1/notes-platform/internal/views"
)

type AdminClassHandler struct {
	classRepo *academic.ClassRepository
	renderer  *views.Renderer
}

func NewAdminClassHandler(
	classRepo *academic.ClassRepository,
	renderer *views.Renderer,
) *AdminClassHandler {
	return &AdminClassHandler{
		classRepo: classRepo,
		renderer:  renderer,
	}
}

type AdminClassesPageData struct {
	Classes []academic.Class
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

	_, err := h.classRepo.Create(r.Context(), academic.CreateClassParams{
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
