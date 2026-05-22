package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
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

type AdminSubjectEditPageData struct {
	Semesters []academic.Semester
	Subject   academic.Subject
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

func (h *AdminSubjectHandler) Edit(w http.ResponseWriter, r *http.Request) {
	subjectID, err := uuid.Parse(chi.URLParam(r, "subjectID"))
	if err != nil {
		http.NotFound(w, r)
		return
	}

	pageData, err := h.editPageData(r, subjectID)
	if err != nil {
		if errors.Is(err, academic.ErrSubjectNotFound) {
			http.NotFound(w, r)
			return
		}

		slog.Error("failed to load subject edit page", "error", err)
		http.Error(w, "Failed to load subject", http.StatusInternalServerError)
		return
	}

	h.renderer.Render(w, r, "admin_subject_edit.tmpl", views.TemplateData{
		Title: "Edit Subject",
		Data:  pageData,
	})
}

func (h *AdminSubjectHandler) Update(w http.ResponseWriter, r *http.Request) {
	subjectID, err := uuid.Parse(chi.URLParam(r, "subjectID"))
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if err := r.ParseForm(); err != nil {
		h.renderEditWithError(w, r, subjectID, "Invalid form submission.")
		return
	}

	semesterIDValue := strings.TrimSpace(r.PostForm.Get("semester_id"))
	name := strings.TrimSpace(r.PostForm.Get("name"))
	description := strings.TrimSpace(r.PostForm.Get("description"))
	sortOrderValue := strings.TrimSpace(r.PostForm.Get("sort_order"))
	isPublished := r.PostForm.Get("is_published") == "on"

	semesterID, err := uuid.Parse(semesterIDValue)
	if err != nil {
		h.renderEditWithError(w, r, subjectID, "Please select a valid semester.")
		return
	}

	sortOrder := 0
	if sortOrderValue != "" {
		parsedSortOrder, err := strconv.Atoi(sortOrderValue)
		if err != nil {
			h.renderEditWithError(w, r, subjectID, "Sort order must be a number.")
			return
		}

		sortOrder = parsedSortOrder
	}

	if name == "" {
		h.renderEditWithError(w, r, subjectID, "Subject name is required.")
		return
	}

	_, err = h.subjectRepo.Update(r.Context(), academic.UpdateSubjectParams{
		ID:          subjectID,
		SemesterID:  semesterID,
		Name:        name,
		Description: description,
		SortOrder:   sortOrder,
		IsPublished: isPublished,
	})
	if err != nil {
		if isUniqueViolation(err) {
			h.renderEditWithError(w, r, subjectID, "This subject already exists for the selected semester.")
			return
		}

		if errors.Is(err, academic.ErrSubjectNotFound) {
			http.NotFound(w, r)
			return
		}

		slog.Error("failed to update subject", "error", err)
		h.renderEditWithError(w, r, subjectID, "Failed to update subject.")
		return
	}

	http.Redirect(w, r, "/admin/subjects", http.StatusSeeOther)
}

func (h *AdminSubjectHandler) renderEditWithError(w http.ResponseWriter, r *http.Request, subjectID uuid.UUID, message string) {
	pageData, err := h.editPageData(r, subjectID)
	if err != nil {
		slog.Error("failed to reload subject edit page after error", "error", err)
		http.Error(w, "Failed to load subject", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusUnprocessableEntity)

	h.renderer.Render(w, r, "admin_subject_edit.tmpl", views.TemplateData{
		Title: "Edit Subject",
		Error: message,
		Data:  pageData,
	})
}

func (h *AdminSubjectHandler) editPageData(r *http.Request, subjectID uuid.UUID) (AdminSubjectEditPageData, error) {
	semesters, err := h.semesterRepo.List(r.Context())
	if err != nil {
		return AdminSubjectEditPageData{}, err
	}

	subjectItem, err := h.subjectRepo.FindByID(r.Context(), subjectID)
	if err != nil {
		return AdminSubjectEditPageData{}, err
	}

	return AdminSubjectEditPageData{
		Semesters: semesters,
		Subject:   subjectItem,
	}, nil
}
