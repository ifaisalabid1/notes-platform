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

type AdminChapterHandler struct {
	unitRepo       *academic.UnitRepository
	chapterRepo    *academic.ChapterRepository
	sessionManager *scs.SessionManager
	auditRepo      *audit.Repository
	renderer       *views.Renderer
}

type AdminChapterEditPageData struct {
	Units   []academic.Unit
	Chapter academic.Chapter
}

func NewAdminChapterHandler(
	unitRepo *academic.UnitRepository,
	chapterRepo *academic.ChapterRepository,
	sessionManager *scs.SessionManager,
	auditRepo *audit.Repository,
	renderer *views.Renderer,
) *AdminChapterHandler {
	return &AdminChapterHandler{
		unitRepo:       unitRepo,
		chapterRepo:    chapterRepo,
		sessionManager: sessionManager,
		auditRepo:      auditRepo,
		renderer:       renderer,
	}
}

type AdminChaptersPageData struct {
	Units    []academic.Unit
	Chapters []academic.Chapter
}

func (h *AdminChapterHandler) Index(w http.ResponseWriter, r *http.Request) {
	pageData, err := h.pageData(r)
	if err != nil {
		slog.Error("failed to load chapters page data", "error", err)
		http.Error(w, "Failed to load chapters", http.StatusInternalServerError)
		return
	}

	h.renderer.Render(w, r, "admin_chapters.tmpl", views.TemplateData{
		Title: "Chapters",
		Data:  pageData,
	})
}

func (h *AdminChapterHandler) Store(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.renderIndexWithError(w, r, "Invalid form submission.")
		return
	}

	unitIDValue := strings.TrimSpace(r.PostForm.Get("unit_id"))
	name := strings.TrimSpace(r.PostForm.Get("name"))
	description := strings.TrimSpace(r.PostForm.Get("description"))
	sortOrderValue := strings.TrimSpace(r.PostForm.Get("sort_order"))
	isPublished := r.PostForm.Get("is_published") == "on"

	unitID, err := uuid.Parse(unitIDValue)
	if err != nil {
		h.renderIndexWithError(w, r, "Please select a valid unit.")
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
		h.renderIndexWithError(w, r, "Chapter name is required.")
		return
	}

	createdChapter, err := h.chapterRepo.Create(r.Context(), academic.CreateChapterParams{
		UnitID:      unitID,
		Name:        name,
		Description: description,
		SortOrder:   sortOrder,
		IsPublished: isPublished,
	})
	if err != nil {
		if isUniqueViolation(err) {
			h.renderIndexWithError(w, r, "This chapter already exists for the selected unit.")
			return
		}

		slog.Error("failed to create chapter", "error", err)
		h.renderIndexWithError(w, r, "Failed to create chapter.")
		return
	}

	writeAuditLog(
		r,
		h.sessionManager,
		h.auditRepo,
		"chapter_created",
		"chapter",
		&createdChapter.ID,
		"Created chapter",
		map[string]any{
			"name":         createdChapter.Name,
			"slug":         createdChapter.Slug,
			"unit_id":      createdChapter.UnitID.String(),
			"is_published": createdChapter.IsPublished,
			"sort_order":   createdChapter.SortOrder,
		},
	)

	http.Redirect(w, r, "/admin/chapters", http.StatusSeeOther)
}

func (h *AdminChapterHandler) renderIndexWithError(w http.ResponseWriter, r *http.Request, message string) {
	pageData, err := h.pageData(r)
	if err != nil {
		slog.Error("failed to load chapters page data after error", "error", err)
		http.Error(w, "Failed to load chapters", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusUnprocessableEntity)

	h.renderer.Render(w, r, "admin_chapters.tmpl", views.TemplateData{
		Title: "Chapters",
		Error: message,
		Data:  pageData,
	})
}

func (h *AdminChapterHandler) pageData(r *http.Request) (AdminChaptersPageData, error) {
	units, err := h.unitRepo.List(r.Context())
	if err != nil {
		return AdminChaptersPageData{}, err
	}

	chapters, err := h.chapterRepo.List(r.Context())
	if err != nil {
		return AdminChaptersPageData{}, err
	}

	return AdminChaptersPageData{
		Units:    units,
		Chapters: chapters,
	}, nil
}

func (h *AdminChapterHandler) Edit(w http.ResponseWriter, r *http.Request) {
	chapterID, err := uuid.Parse(chi.URLParam(r, "chapterID"))
	if err != nil {
		http.NotFound(w, r)
		return
	}

	pageData, err := h.editPageData(r, chapterID)
	if err != nil {
		if errors.Is(err, academic.ErrChapterNotFound) {
			http.NotFound(w, r)
			return
		}

		slog.Error("failed to load chapter edit page", "error", err)
		http.Error(w, "Failed to load chapter", http.StatusInternalServerError)
		return
	}

	h.renderer.Render(w, r, "admin_chapter_edit.tmpl", views.TemplateData{
		Title: "Edit Chapter",
		Data:  pageData,
	})
}

func (h *AdminChapterHandler) Update(w http.ResponseWriter, r *http.Request) {
	chapterID, err := uuid.Parse(chi.URLParam(r, "chapterID"))
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if err := r.ParseForm(); err != nil {
		h.renderEditWithError(w, r, chapterID, "Invalid form submission.")
		return
	}

	unitIDValue := strings.TrimSpace(r.PostForm.Get("unit_id"))
	name := strings.TrimSpace(r.PostForm.Get("name"))
	description := strings.TrimSpace(r.PostForm.Get("description"))
	sortOrderValue := strings.TrimSpace(r.PostForm.Get("sort_order"))
	isPublished := r.PostForm.Get("is_published") == "on"

	unitID, err := uuid.Parse(unitIDValue)
	if err != nil {
		h.renderEditWithError(w, r, chapterID, "Please select a valid unit.")
		return
	}

	sortOrder := 0
	if sortOrderValue != "" {
		parsedSortOrder, err := strconv.Atoi(sortOrderValue)
		if err != nil {
			h.renderEditWithError(w, r, chapterID, "Sort order must be a number.")
			return
		}

		sortOrder = parsedSortOrder
	}

	if name == "" {
		h.renderEditWithError(w, r, chapterID, "Chapter name is required.")
		return
	}

	updatedChapter, err := h.chapterRepo.Update(r.Context(), academic.UpdateChapterParams{
		ID:          chapterID,
		UnitID:      unitID,
		Name:        name,
		Description: description,
		SortOrder:   sortOrder,
		IsPublished: isPublished,
	})
	if err != nil {
		if isUniqueViolation(err) {
			h.renderEditWithError(w, r, chapterID, "This chapter already exists for the selected unit.")
			return
		}

		if errors.Is(err, academic.ErrChapterNotFound) {
			http.NotFound(w, r)
			return
		}

		slog.Error("failed to update chapter", "error", err)
		h.renderEditWithError(w, r, chapterID, "Failed to update chapter.")
		return
	}

	writeAuditLog(
		r,
		h.sessionManager,
		h.auditRepo,
		"chapter_updated",
		"chapter",
		&updatedChapter.ID,
		"Updated chapter",
		map[string]any{
			"name":         updatedChapter.Name,
			"slug":         updatedChapter.Slug,
			"unit_id":      updatedChapter.UnitID.String(),
			"is_published": updatedChapter.IsPublished,
			"sort_order":   updatedChapter.SortOrder,
		},
	)

	http.Redirect(w, r, "/admin/chapters", http.StatusSeeOther)
}

func (h *AdminChapterHandler) renderEditWithError(w http.ResponseWriter, r *http.Request, chapterID uuid.UUID, message string) {
	pageData, err := h.editPageData(r, chapterID)
	if err != nil {
		slog.Error("failed to reload chapter edit page after error", "error", err)
		http.Error(w, "Failed to load chapter", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusUnprocessableEntity)

	h.renderer.Render(w, r, "admin_chapter_edit.tmpl", views.TemplateData{
		Title: "Edit Chapter",
		Error: message,
		Data:  pageData,
	})
}

func (h *AdminChapterHandler) editPageData(r *http.Request, chapterID uuid.UUID) (AdminChapterEditPageData, error) {
	units, err := h.unitRepo.List(r.Context())
	if err != nil {
		return AdminChapterEditPageData{}, err
	}

	chapterItem, err := h.chapterRepo.FindByID(r.Context(), chapterID)
	if err != nil {
		return AdminChapterEditPageData{}, err
	}

	return AdminChapterEditPageData{
		Units:   units,
		Chapter: chapterItem,
	}, nil
}
