package handlers

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/ifaisalabid1/notes-platform/internal/academic"
	"github.com/ifaisalabid1/notes-platform/internal/fileproxy"
	"github.com/ifaisalabid1/notes-platform/internal/views"
)

type PublicHandler struct {
	publicRepo      *academic.PublicRepository
	noteRepo        *academic.NoteRepository
	fileProxySigner *fileproxy.Signer
	renderer        *views.Renderer
}

func NewPublicHandler(
	publicRepo *academic.PublicRepository,
	noteRepo *academic.NoteRepository,
	fileProxySigner *fileproxy.Signer,
	renderer *views.Renderer,
) *PublicHandler {
	return &PublicHandler{
		publicRepo:      publicRepo,
		noteRepo:        noteRepo,
		fileProxySigner: fileProxySigner,
		renderer:        renderer,
	}
}

type PublicClassesPageData struct {
	Classes     []academic.Class
	LatestNotes []PublicLatestNoteItem
	Galleries   []PublicGallerySubjectItem
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

type PublicLatestNoteItem struct {
	Note    academic.Note
	FileURL string
}

type PublicGallerySubjectItem struct {
	SubjectID    uuid.UUID
	SubjectName  string
	ClassName    string
	SemesterName string
	ChapterCount int
	PhotoCount   int
	CoverPhoto   PublicGalleryPhoto
	URL          string
}

type PublicGalleryChapterItem struct {
	ChapterID   uuid.UUID
	ChapterName string
	UnitName    string
	PhotoCount  int
	CoverPhoto  PublicGalleryPhoto
	URL         string
}

type PublicGalleryPhoto struct {
	Title   string
	FileURL string
}

type PublicGallerySubjectPageData struct {
	SubjectName  string
	ClassName    string
	SemesterName string
	Chapters     []PublicGalleryChapterItem
}

type PublicGalleryChapterPageData struct {
	Class    academic.Class
	Semester academic.Semester
	Subject  academic.Subject
	Unit     academic.Unit
	Chapter  academic.Chapter
	Photos   []PublicGalleryPhoto
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

	latestNotes, err := h.publicRepo.LatestPublishedNotes(r.Context(), 6)
	if err != nil {
		slog.Error("failed to load latest public notes", "error", err)
		http.Error(w, "Failed to load latest notes", http.StatusInternalServerError)
		return
	}

	latestNoteItems := make([]PublicLatestNoteItem, 0, len(latestNotes))

	for _, note := range latestNotes {
		fileURL, err := h.fileProxySigner.SignedFileURL(note.StorageKey)
		if err != nil {
			slog.Error("failed to sign latest note url", "error", err)
			http.Error(w, "Failed to load latest note links", http.StatusInternalServerError)
			return
		}

		latestNoteItems = append(latestNoteItems, PublicLatestNoteItem{
			Note:    note,
			FileURL: fileURL,
		})
	}

	galleries, err := h.homeGallerySubjects(r, 11)
	if err != nil {
		slog.Error("failed to load homepage galleries", "error", err)
		http.Error(w, "Failed to load galleries", http.StatusInternalServerError)
		return
	}

	var styles []string
	var scripts []string

	if len(galleries) > 0 {
		styles = []string{
			"/static/vendor/swiper/swiper-bundle.min.css",
		}
		scripts = []string{
			"/static/vendor/swiper/swiper-bundle.min.js",
			"/static/js/home-gallery.js",
		}
	}

	h.renderer.Render(w, r, "public_classes.tmpl", views.TemplateData{
		Title:       "Browse Classes",
		Description: "Browse class notes, semesters, subjects, units, chapters, and recently published study materials.",
		Styles:      styles,
		Scripts:     scripts,
		Data: PublicClassesPageData{
			Classes:     classes,
			LatestNotes: latestNoteItems,
			Galleries:   galleries,
		},
	})
}

func (h *PublicHandler) homeGallerySubjects(r *http.Request, limit int) ([]PublicGallerySubjectItem, error) {
	subjects, err := h.publicRepo.PublishedImageGallerySubjects(r.Context(), limit)
	if err != nil {
		return nil, err
	}

	items := make([]PublicGallerySubjectItem, 0, len(subjects))

	for _, subject := range subjects {
		fileURL, err := h.fileProxySigner.SignedFileURL(subject.CoverNote.StorageKey)
		if err != nil {
			return nil, fmt.Errorf("sign gallery subject cover url: %w", err)
		}

		items = append(items, PublicGallerySubjectItem{
			SubjectID:    subject.SubjectID,
			SubjectName:  subject.SubjectName,
			ClassName:    subject.ClassName,
			SemesterName: subject.SemesterName,
			ChapterCount: subject.ChapterCount,
			PhotoCount:   subject.PhotoCount,
			CoverPhoto: PublicGalleryPhoto{
				Title:   subject.CoverNote.Title,
				FileURL: fileURL,
			},
			URL: "/gallery/subjects/" + subject.SubjectID.String(),
		})
	}

	return items, nil
}

func (h *PublicHandler) GallerySubject(w http.ResponseWriter, r *http.Request) {
	subjectID, err := uuid.Parse(chi.URLParam(r, "subjectID"))
	if err != nil {
		http.NotFound(w, r)
		return
	}

	chapters, err := h.publicRepo.PublishedImageGalleryChaptersBySubjectID(r.Context(), subjectID)
	if err != nil {
		slog.Error("failed to load public gallery subject chapters", "error", err)
		http.Error(w, "Failed to load gallery chapters", http.StatusInternalServerError)
		return
	}

	if len(chapters) == 0 {
		http.NotFound(w, r)
		return
	}

	items := make([]PublicGalleryChapterItem, 0, len(chapters))

	for _, chapter := range chapters {
		fileURL, err := h.fileProxySigner.SignedFileURL(chapter.CoverNote.StorageKey)
		if err != nil {
			slog.Error("failed to sign gallery chapter cover url", "error", err)
			http.Error(w, "Failed to load gallery chapter links", http.StatusInternalServerError)
			return
		}

		items = append(items, PublicGalleryChapterItem{
			ChapterID:   chapter.ChapterID,
			ChapterName: chapter.ChapterName,
			UnitName:    chapter.UnitName,
			PhotoCount:  chapter.PhotoCount,
			CoverPhoto: PublicGalleryPhoto{
				Title:   chapter.CoverNote.Title,
				FileURL: fileURL,
			},
			URL: "/gallery/chapters/" + chapter.ChapterID.String(),
		})
	}

	first := chapters[0]

	h.renderer.Render(w, r, "public_gallery_subject.tmpl", views.TemplateData{
		Title:       fmt.Sprintf("%s Gallery", first.SubjectName),
		Description: fmt.Sprintf("Browse photo gallery chapters for %s.", first.SubjectName),
		Data: PublicGallerySubjectPageData{
			SubjectName:  first.SubjectName,
			ClassName:    first.ClassName,
			SemesterName: first.SemesterName,
			Chapters:     items,
		},
	})
}

func (h *PublicHandler) GalleryChapter(w http.ResponseWriter, r *http.Request) {
	chapterID, err := uuid.Parse(chi.URLParam(r, "chapterID"))
	if err != nil {
		http.NotFound(w, r)
		return
	}

	classItem, semesterItem, subjectItem, unitItem, chapterItem, notes, err := h.publicRepo.PublishedImageGalleryNotesByChapterID(r.Context(), chapterID)
	if err != nil {
		slog.Error("failed to load public gallery chapter photos", "error", err)
		http.NotFound(w, r)
		return
	}

	photos := make([]PublicGalleryPhoto, 0, len(notes))

	for _, note := range notes {
		fileURL, err := h.fileProxySigner.SignedFileURL(note.StorageKey)
		if err != nil {
			slog.Error("failed to sign gallery chapter photo url", "error", err)
			http.Error(w, "Failed to load gallery photos", http.StatusInternalServerError)
			return
		}

		photos = append(photos, PublicGalleryPhoto{
			Title:   note.Title,
			FileURL: fileURL,
		})
	}

	h.renderer.Render(w, r, "public_gallery_chapter.tmpl", views.TemplateData{
		Title:       fmt.Sprintf("%s Gallery", chapterItem.Name),
		Description: fmt.Sprintf("View gallery photos for %s in %s.", chapterItem.Name, subjectItem.Name),
		Styles: []string{
			"/static/vendor/glightbox/glightbox.min.css",
		},
		Scripts: []string{
			"/static/vendor/glightbox/glightbox.min.js",
			"/static/js/gallery-lightbox.js",
		},
		Data: PublicGalleryChapterPageData{
			Class:    classItem,
			Semester: semesterItem,
			Subject:  subjectItem,
			Unit:     unitItem,
			Chapter:  chapterItem,
			Photos:   photos,
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

func (h *PublicHandler) ViewNote(w http.ResponseWriter, r *http.Request) {
	noteID, err := uuid.Parse(chi.URLParam(r, "noteID"))
	if err != nil {
		http.NotFound(w, r)
		return
	}

	noteItem, err := h.publicRepo.PublishedNoteByID(r.Context(), noteID)
	if err != nil {
		if errors.Is(err, academic.ErrPublicNoteNotFound) {
			http.NotFound(w, r)
			return
		}

		slog.Error("failed to load public note for view", "error", err)
		http.Error(w, "Failed to load note", http.StatusInternalServerError)
		return
	}

	if err := h.noteRepo.IncrementViewCount(r.Context(), noteItem.ID); err != nil {
		slog.Error(
			"failed to increment note view count",
			"note_id", noteItem.ID.String(),
			"error", err,
		)
	}

	fileURL, err := h.fileProxySigner.SignedFileURL(noteItem.StorageKey)
	if err != nil {
		slog.Error("failed to sign public note file url", "error", err)
		http.Error(w, "Failed to open note", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fileURL, http.StatusSeeOther)
}
