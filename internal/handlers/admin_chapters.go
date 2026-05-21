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

type AdminChapterHandler struct {
	unitRepo    *academic.UnitRepository
	chapterRepo *academic.ChapterRepository
	renderer    *views.Renderer
}

func NewAdminChapterHandler(
	unitRepo *academic.UnitRepository,
	chapterRepo *academic.ChapterRepository,
	renderer *views.Renderer,
) *AdminChapterHandler {
	return &AdminChapterHandler{
		unitRepo:    unitRepo,
		chapterRepo: chapterRepo,
		renderer:    renderer,
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

	_, err = h.chapterRepo.Create(r.Context(), academic.CreateChapterParams{
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
