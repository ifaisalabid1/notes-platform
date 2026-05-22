package handlers

import (
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"github.com/ifaisalabid1/notes-platform/internal/academic"
	"github.com/ifaisalabid1/notes-platform/internal/views"
)

type AdminHTMXHandler struct {
	publicRepo *academic.PublicRepository
	renderer   *views.Renderer
}

func NewAdminHTMXHandler(
	publicRepo *academic.PublicRepository,
	renderer *views.Renderer,
) *AdminHTMXHandler {
	return &AdminHTMXHandler{
		publicRepo: publicRepo,
		renderer:   renderer,
	}
}

type AdminHTMXSemestersData struct {
	Semesters []academic.Semester
}

type AdminHTMXSubjectsData struct {
	Subjects []academic.Subject
}

type AdminHTMXUnitsData struct {
	Units []academic.Unit
}

type AdminHTMXChaptersData struct {
	Chapters []academic.Chapter
}

func (h *AdminHTMXHandler) SemestersByClass(w http.ResponseWriter, r *http.Request) {
	classID, err := uuid.Parse(r.URL.Query().Get("class_id"))
	if err != nil {
		h.renderer.RenderPartial(w, r, "partials_empty_select.tmpl", views.TemplateData{})
		return
	}

	semesters, err := h.publicRepo.SemestersByClassID(r.Context(), classID)
	if err != nil {
		slog.Error("failed to load htmx semesters", "error", err)
		http.Error(w, "Failed to load semesters", http.StatusInternalServerError)
		return
	}

	h.renderer.RenderPartial(w, r, "partials_semester_select.tmpl", views.TemplateData{
		Data: AdminHTMXSemestersData{
			Semesters: semesters,
		},
	})
}

func (h *AdminHTMXHandler) SubjectsBySemester(w http.ResponseWriter, r *http.Request) {
	semesterID, err := uuid.Parse(r.URL.Query().Get("semester_id"))
	if err != nil {
		h.renderer.RenderPartial(w, r, "partials_empty_select.tmpl", views.TemplateData{})
		return
	}

	subjects, err := h.publicRepo.SubjectsBySemesterID(r.Context(), semesterID)
	if err != nil {
		slog.Error("failed to load htmx subjects", "error", err)
		http.Error(w, "Failed to load subjects", http.StatusInternalServerError)
		return
	}

	h.renderer.RenderPartial(w, r, "partials_subject_select.tmpl", views.TemplateData{
		Data: AdminHTMXSubjectsData{
			Subjects: subjects,
		},
	})
}

func (h *AdminHTMXHandler) UnitsBySubject(w http.ResponseWriter, r *http.Request) {
	subjectID, err := uuid.Parse(r.URL.Query().Get("subject_id"))
	if err != nil {
		h.renderer.RenderPartial(w, r, "partials_empty_select.tmpl", views.TemplateData{})
		return
	}

	units, err := h.publicRepo.UnitsBySubjectID(r.Context(), subjectID)
	if err != nil {
		slog.Error("failed to load htmx units", "error", err)
		http.Error(w, "Failed to load units", http.StatusInternalServerError)
		return
	}

	h.renderer.RenderPartial(w, r, "partials_unit_select.tmpl", views.TemplateData{
		Data: AdminHTMXUnitsData{
			Units: units,
		},
	})
}

func (h *AdminHTMXHandler) ChaptersByUnit(w http.ResponseWriter, r *http.Request) {
	unitID, err := uuid.Parse(r.URL.Query().Get("unit_id"))
	if err != nil {
		h.renderer.RenderPartial(w, r, "partials_empty_select.tmpl", views.TemplateData{})
		return
	}

	chapters, err := h.publicRepo.ChaptersByUnitID(r.Context(), unitID)
	if err != nil {
		slog.Error("failed to load htmx chapters", "error", err)
		http.Error(w, "Failed to load chapters", http.StatusInternalServerError)
		return
	}

	h.renderer.RenderPartial(w, r, "partials_chapter_select.tmpl", views.TemplateData{
		Data: AdminHTMXChaptersData{
			Chapters: chapters,
		},
	})
}
