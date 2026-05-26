package handlers

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/ifaisalabid1/notes-platform/internal/academic"
	"github.com/ifaisalabid1/notes-platform/internal/fileproxy"
	"github.com/ifaisalabid1/notes-platform/internal/views"
)

type PublicHandler struct {
	publicRepo      *academic.PublicRepository
	fileProxySigner *fileproxy.Signer
	renderer        *views.Renderer
}

func NewPublicHandler(
	publicRepo *academic.PublicRepository,
	fileProxySigner *fileproxy.Signer,
	renderer *views.Renderer,
) *PublicHandler {
	return &PublicHandler{
		publicRepo:      publicRepo,
		fileProxySigner: fileProxySigner,
		renderer:        renderer,
	}
}

type PublicClassesPageData struct {
	Classes []academic.Class
}

type PublicSemestersPageData struct {
	Class     academic.Class
	Semesters []academic.Semester
}

type PublicSubjectsPageData struct {
	Class    academic.Class
	Semester academic.Semester
	Subjects []academic.Subject
}

type PublicUnitsPageData struct {
	Class    academic.Class
	Semester academic.Semester
	Subject  academic.Subject
	Units    []academic.Unit
}

type PublicChaptersPageData struct {
	Class    academic.Class
	Semester academic.Semester
	Subject  academic.Subject
	Unit     academic.Unit
	Chapters []academic.Chapter
}

type PublicNoteItem struct {
	Note    academic.Note
	FileURL string
}

type PublicNotesPageData struct {
	Class    academic.Class
	Semester academic.Semester
	Subject  academic.Subject
	Unit     academic.Unit
	Chapter  academic.Chapter
	Notes    []PublicNoteItem
}

type PublicSearchItem struct {
	Note    academic.Note
	FileURL string
}

type PublicSearchPagination struct {
	Query       string
	Page        int
	PerPage     int
	TotalCount  int
	TotalPages  int
	HasPrevious bool
	HasNext     bool
	PreviousURL string
	NextURL     string
}

type PublicSearchPageData struct {
	Results    []PublicSearchItem
	Pagination PublicSearchPagination
}

func (h *PublicHandler) Home(w http.ResponseWriter, r *http.Request) {
	classes, err := h.publicRepo.PublishedClasses(r.Context())
	if err != nil {
		slog.Error("failed to load public classes", "error", err)
		http.Error(w, "Failed to load classes", http.StatusInternalServerError)
		return
	}

	h.renderer.Render(w, r, "public_classes.tmpl", views.TemplateData{
		Title:       "Browse Classes",
		Description: "Browse class notes, semesters, subjects, units, chapters, and study materials.",
		Data: PublicClassesPageData{
			Classes: classes,
		},
	})
}

func (h *PublicHandler) Semesters(w http.ResponseWriter, r *http.Request) {
	classSlug := chi.URLParam(r, "classSlug")

	classItem, semesters, err := h.publicRepo.PublishedSemestersByClassSlug(r.Context(), classSlug)
	if err != nil {
		slog.Error("failed to load public semesters", "error", err)
		http.NotFound(w, r)
		return
	}

	h.renderer.Render(w, r, "public_semesters.tmpl", views.TemplateData{
		Title:       classItem.Name,
		Description: fmt.Sprintf("Browse semesters and notes for %s.", classItem.Name),
		Data: PublicSemestersPageData{
			Class:     classItem,
			Semesters: semesters,
		},
	})
}

func (h *PublicHandler) Subjects(w http.ResponseWriter, r *http.Request) {
	classSlug := chi.URLParam(r, "classSlug")
	semesterSlug := chi.URLParam(r, "semesterSlug")

	classItem, semesterItem, subjects, err := h.publicRepo.PublishedSubjects(r.Context(), classSlug, semesterSlug)
	if err != nil {
		slog.Error("failed to load public subjects", "error", err)
		http.NotFound(w, r)
		return
	}

	h.renderer.Render(w, r, "public_subjects.tmpl", views.TemplateData{
		Title:       semesterItem.Name,
		Description: fmt.Sprintf("Browse subjects for %s in %s.", semesterItem.Name, classItem.Name),
		Data: PublicSubjectsPageData{
			Class:    classItem,
			Semester: semesterItem,
			Subjects: subjects,
		},
	})
}

func (h *PublicHandler) Units(w http.ResponseWriter, r *http.Request) {
	classSlug := chi.URLParam(r, "classSlug")
	semesterSlug := chi.URLParam(r, "semesterSlug")
	subjectSlug := chi.URLParam(r, "subjectSlug")

	classItem, semesterItem, subjectItem, units, err := h.publicRepo.PublishedUnits(r.Context(), classSlug, semesterSlug, subjectSlug)
	if err != nil {
		slog.Error("failed to load public units", "error", err)
		http.NotFound(w, r)
		return
	}

	h.renderer.Render(w, r, "public_units.tmpl", views.TemplateData{
		Title:       subjectItem.Name,
		Description: fmt.Sprintf("Browse units and notes for %s.", subjectItem.Name),
		Data: PublicUnitsPageData{
			Class:    classItem,
			Semester: semesterItem,
			Subject:  subjectItem,
			Units:    units,
		},
	})
}

func (h *PublicHandler) Chapters(w http.ResponseWriter, r *http.Request) {
	classSlug := chi.URLParam(r, "classSlug")
	semesterSlug := chi.URLParam(r, "semesterSlug")
	subjectSlug := chi.URLParam(r, "subjectSlug")
	unitSlug := chi.URLParam(r, "unitSlug")

	classItem, semesterItem, subjectItem, unitItem, chapters, err := h.publicRepo.PublishedChapters(r.Context(), classSlug, semesterSlug, subjectSlug, unitSlug)
	if err != nil {
		slog.Error("failed to load public chapters", "error", err)
		http.NotFound(w, r)
		return
	}

	h.renderer.Render(w, r, "public_chapters.tmpl", views.TemplateData{
		Title:       unitItem.Name,
		Description: fmt.Sprintf("Browse chapters and notes for %s.", unitItem.Name),
		Data: PublicChaptersPageData{
			Class:    classItem,
			Semester: semesterItem,
			Subject:  subjectItem,
			Unit:     unitItem,
			Chapters: chapters,
		},
	})
}

func (h *PublicHandler) Notes(w http.ResponseWriter, r *http.Request) {
	classSlug := chi.URLParam(r, "classSlug")
	semesterSlug := chi.URLParam(r, "semesterSlug")
	subjectSlug := chi.URLParam(r, "subjectSlug")
	unitSlug := chi.URLParam(r, "unitSlug")
	chapterSlug := chi.URLParam(r, "chapterSlug")

	classItem, semesterItem, subjectItem, unitItem, chapterItem, notes, err := h.publicRepo.PublishedNotes(
		r.Context(),
		classSlug,
		semesterSlug,
		subjectSlug,
		unitSlug,
		chapterSlug,
	)
	if err != nil {
		slog.Error("failed to load public notes", "error", err)
		http.NotFound(w, r)
		return
	}

	noteItems := make([]PublicNoteItem, 0, len(notes))

	for _, note := range notes {
		fileURL, err := h.fileProxySigner.SignedFileURL(note.StorageKey)
		if err != nil {
			slog.Error("failed to sign public note url", "error", err)
			http.Error(w, "Failed to load note links", http.StatusInternalServerError)
			return
		}

		if err := h.publicRepo.IncrementNoteViewCount(r.Context(), note.ID); err != nil {
			slog.Error("failed to increment note view count", "error", err)
		}

		noteItems = append(noteItems, PublicNoteItem{
			Note:    note,
			FileURL: fileURL,
		})
	}

	h.renderer.Render(w, r, "public_notes.tmpl", views.TemplateData{
		Title:       fmt.Sprintf("%s Notes", chapterItem.Name),
		Description: fmt.Sprintf("View notes for %s in %s.", chapterItem.Name, subjectItem.Name),
		Data: PublicNotesPageData{
			Class:    classItem,
			Semester: semesterItem,
			Subject:  subjectItem,
			Unit:     unitItem,
			Chapter:  chapterItem,
			Notes:    noteItems,
		},
	})
}

func (h *PublicHandler) Search(w http.ResponseWriter, r *http.Request) {
	queryValues := r.URL.Query()

	searchQuery := strings.TrimSpace(queryValues.Get("q"))
	page := parsePositiveInt(queryValues.Get("page"), 1)
	perPage := parsePositiveInt(queryValues.Get("per_page"), 20)

	if perPage > 50 {
		perPage = 50
	}

	offset := (page - 1) * perPage

	searchResult, err := h.publicRepo.SearchPublishedNotes(r.Context(), academic.PublicNoteSearchParams{
		Query:  searchQuery,
		Limit:  perPage,
		Offset: offset,
	})
	if err != nil {
		slog.Error("failed to search public notes", "error", err)
		http.Error(w, "Failed to search notes", http.StatusInternalServerError)
		return
	}

	totalPages := 0
	if searchResult.TotalCount > 0 {
		totalPages = (searchResult.TotalCount + perPage - 1) / perPage
	}

	if totalPages > 0 && page > totalPages {
		page = totalPages
		offset = (page - 1) * perPage

		searchResult, err = h.publicRepo.SearchPublishedNotes(r.Context(), academic.PublicNoteSearchParams{
			Query:  searchQuery,
			Limit:  perPage,
			Offset: offset,
		})
		if err != nil {
			slog.Error("failed to search public notes after page correction", "error", err)
			http.Error(w, "Failed to search notes", http.StatusInternalServerError)
			return
		}
	}

	items := make([]PublicSearchItem, 0, len(searchResult.Notes))

	for _, note := range searchResult.Notes {
		fileURL, err := h.fileProxySigner.SignedFileURL(note.StorageKey)
		if err != nil {
			slog.Error("failed to sign public search note url", "error", err)
			http.Error(w, "Failed to load note links", http.StatusInternalServerError)
			return
		}

		items = append(items, PublicSearchItem{
			Note:    note,
			FileURL: fileURL,
		})
	}

	hasPrevious := page > 1
	hasNext := totalPages > 0 && page < totalPages

	previousURL := ""
	if hasPrevious {
		previousURL = buildPublicSearchURL(searchQuery, page-1, perPage)
	}

	nextURL := ""
	if hasNext {
		nextURL = buildPublicSearchURL(searchQuery, page+1, perPage)
	}

	title := "Search Notes"
	description := "Search published classroom notes and study materials."

	if searchQuery != "" {
		title = fmt.Sprintf("Search results for %s", searchQuery)
		description = fmt.Sprintf("Search results for %s in published classroom notes.", searchQuery)
	}

	h.renderer.Render(w, r, "public_search.tmpl", views.TemplateData{
		Title:       title,
		Description: description,
		Data: PublicSearchPageData{
			Results: items,
			Pagination: PublicSearchPagination{
				Query:       searchQuery,
				Page:        page,
				PerPage:     perPage,
				TotalCount:  searchResult.TotalCount,
				TotalPages:  totalPages,
				HasPrevious: hasPrevious,
				HasNext:     hasNext,
				PreviousURL: previousURL,
				NextURL:     nextURL,
			},
		},
	})
}

func buildPublicSearchURL(search string, page int, perPage int) string {
	values := url.Values{}

	if strings.TrimSpace(search) != "" {
		values.Set("q", strings.TrimSpace(search))
	}

	values.Set("page", strconv.Itoa(page))
	values.Set("per_page", strconv.Itoa(perPage))

	return "/search?" + values.Encode()
}
