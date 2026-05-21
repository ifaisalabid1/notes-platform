package handlers

import (
	"fmt"
	"log/slog"
	"net/http"

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

func (h *PublicHandler) Home(w http.ResponseWriter, r *http.Request) {
	classes, err := h.publicRepo.PublishedClasses(r.Context())
	if err != nil {
		slog.Error("failed to load public classes", "error", err)
		http.Error(w, "Failed to load classes", http.StatusInternalServerError)
		return
	}

	h.renderer.Render(w, r, "public_classes.tmpl", views.TemplateData{
		Title: "Classes",
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
		Title: classItem.Name,
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
		Title: semesterItem.Name,
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
		Title: subjectItem.Name,
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
		Title: unitItem.Name,
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
		Title: fmt.Sprintf("%s Notes", chapterItem.Name),
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
