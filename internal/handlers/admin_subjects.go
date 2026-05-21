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

type AdminSubjectHandler struct {
	semesterRepo *academic.SemesterRepository
	subjectRepo  *academic.SubjectRepository
	renderer     *views.Renderer
}

func NewAdminSubjectHandler(
	semesterRepo *academic.SemesterRepository,
	subjectRepo *academic.SubjectRepository,
	renderer *views.Renderer,
) *AdminSubjectHandler {
	return &AdminSubjectHandler{
		semesterRepo: semesterRepo,
		subjectRepo:  subjectRepo,
		renderer:     renderer,
	}
}

type AdminSubjectsPageData struct {
	Semesters []academic.Semester
	Subjects  []academic.Subject
}

func (h *AdminSubjectHandler) Index(w http.ResponseWriter, r *http.Request) {
	pageData, err := h.pageData(r)
	if err != nil {
		slog.Error("failed to load subjects page data", "error", err)
		http.Error(w, "Failed to load subjects", http.StatusInternalServerError)
		return
	}

	h.renderer.Render(w, r, "admin_subjects.tmpl", views.TemplateData{
		Title: "Subjects",
		Data:  pageData,
	})
}

func (h *AdminSubjectHandler) Store(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.renderIndexWithError(w, r, "Invalid form submission.")
		return
	}

	semesterIDValue := strings.TrimSpace(r.PostForm.Get("semester_id"))
	name := strings.TrimSpace(r.PostForm.Get("name"))
	description := strings.TrimSpace(r.PostForm.Get("description"))
	sortOrderValue := strings.TrimSpace(r.PostForm.Get("sort_order"))
	isPublished := r.PostForm.Get("is_published") == "on"

	semesterID, err := uuid.Parse(semesterIDValue)
	if err != nil {
		h.renderIndexWithError(w, r, "Please select a valid semester.")
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
		h.renderIndexWithError(w, r, "Subject name is required.")
		return
	}

	_, err = h.subjectRepo.Create(r.Context(), academic.CreateSubjectParams{
		SemesterID:  semesterID,
		Name:        name,
		Description: description,
		SortOrder:   sortOrder,
		IsPublished: isPublished,
	})
	if err != nil {
		if isUniqueViolation(err) {
			h.renderIndexWithError(w, r, "This subject already exists for the selected semester.")
			return
		}

		slog.Error("failed to create subject", "error", err)
		h.renderIndexWithError(w, r, "Failed to create subject.")
		return
	}

	http.Redirect(w, r, "/admin/subjects", http.StatusSeeOther)
}

func (h *AdminSubjectHandler) renderIndexWithError(w http.ResponseWriter, r *http.Request, message string) {
	pageData, err := h.pageData(r)
	if err != nil {
		slog.Error("failed to load subjects page data after error", "error", err)
		http.Error(w, "Failed to load subjects", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusUnprocessableEntity)

	h.renderer.Render(w, r, "admin_subjects.tmpl", views.TemplateData{
		Title: "Subjects",
		Error: message,
		Data:  pageData,
	})
}

func (h *AdminSubjectHandler) pageData(r *http.Request) (AdminSubjectsPageData, error) {
	semesters, err := h.semesterRepo.List(r.Context())
	if err != nil {
		return AdminSubjectsPageData{}, err
	}

	subjects, err := h.subjectRepo.List(r.Context())
	if err != nil {
		return AdminSubjectsPageData{}, err
	}

	return AdminSubjectsPageData{
		Semesters: semesters,
		Subjects:  subjects,
	}, nil
}
