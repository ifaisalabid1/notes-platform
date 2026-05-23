package handlers

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/ifaisalabid1/notes-platform/internal/academic"
	"github.com/ifaisalabid1/notes-platform/internal/fileproxy"
	"github.com/ifaisalabid1/notes-platform/internal/storage"
	"github.com/ifaisalabid1/notes-platform/internal/uploads"
	"github.com/ifaisalabid1/notes-platform/internal/views"
	"github.com/ifaisalabid1/notes-platform/internal/watermark"
)

type AdminNoteHandler struct {
	classRepo       *academic.ClassRepository
	chapterRepo     *academic.ChapterRepository
	noteRepo        *academic.NoteRepository
	r2              *storage.R2Client
	pdfWatermarker  *watermark.PDFWatermarker
	fileProxySigner *fileproxy.Signer
	sessionManager  *scs.SessionManager
	renderer        *views.Renderer
}

func NewAdminNoteHandler(
	classRepo *academic.ClassRepository,
	chapterRepo *academic.ChapterRepository,
	noteRepo *academic.NoteRepository,
	r2 *storage.R2Client,
	pdfWatermarker *watermark.PDFWatermarker,
	fileProxySigner *fileproxy.Signer,
	sessionManager *scs.SessionManager,
	renderer *views.Renderer,
) *AdminNoteHandler {
	return &AdminNoteHandler{
		classRepo:       classRepo,
		chapterRepo:     chapterRepo,
		noteRepo:        noteRepo,
		r2:              r2,
		pdfWatermarker:  pdfWatermarker,
		fileProxySigner: fileProxySigner,
		sessionManager:  sessionManager,
		renderer:        renderer,
	}
}

type AdminNoteListItem struct {
	Note    academic.Note
	FileURL string
}

type AdminNotesPageData struct {
	Classes []academic.Class
	Notes   []AdminNoteListItem
}

type AdminNoteEditPageData struct {
	Chapters []academic.Chapter
	Note     academic.Note
	FileURL  string
}

func (h *AdminNoteHandler) Index(w http.ResponseWriter, r *http.Request) {
	pageData, err := h.pageData(r)
	if err != nil {
		slog.Error("failed to load notes page data", "error", err)
		http.Error(w, "Failed to load notes", http.StatusInternalServerError)
		return
	}

	h.renderer.Render(w, r, "admin_notes.tmpl", views.TemplateData{
		Title: "Notes",
		Data:  pageData,
	})
}

func (h *AdminNoteHandler) Store(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(uploads.MaxUploadSizeBytes); err != nil {
		h.renderIndexWithError(w, r, "Invalid upload. File may be too large.")
		return
	}

	chapterIDValue := strings.TrimSpace(r.PostForm.Get("chapter_id"))
	title := strings.TrimSpace(r.PostForm.Get("title"))
	description := strings.TrimSpace(r.PostForm.Get("description"))
	sortOrderValue := strings.TrimSpace(r.PostForm.Get("sort_order"))
	isPublished := r.PostForm.Get("is_published") == "on"

	chapterID, err := uuid.Parse(chapterIDValue)
	if err != nil {
		h.renderIndexWithError(w, r, "Please select a valid chapter.")
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

	if title == "" {
		h.renderIndexWithError(w, r, "Note title is required.")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		h.renderIndexWithError(w, r, "Please choose a file.")
		return
	}
	defer file.Close()

	validatedFile, err := uploads.ValidateUploadedFile(file, header)
	if err != nil {
		h.renderIndexWithError(w, r, err.Error())
		return
	}

	uploadedByString := h.sessionManager.GetString(r.Context(), "admin_id")
	uploadedBy, err := uuid.Parse(uploadedByString)
	if err != nil {
		slog.Error("failed to parse logged-in admin id", "error", err)
		http.Error(w, "Invalid session", http.StatusUnauthorized)
		return
	}

	noteID := uuid.New()
	storedFileName := buildStoredFileName(noteID, validatedFile.OriginalFileName)
	storageKey := buildNoteStorageKey(chapterID, noteID, storedFileName)

	fileData := validatedFile.Data
	fileSizeBytes := validatedFile.SizeBytes
	isWatermarked := false

	if validatedFile.IsPDF {
		watermarkedPDF, err := h.pdfWatermarker.Apply(validatedFile.Data)
		if err != nil {
			slog.Error("failed to watermark pdf", "error", err)
			h.renderIndexWithError(w, r, "Failed to watermark PDF.")
			return
		}

		fileData = watermarkedPDF.Data
		fileSizeBytes = int64(len(watermarkedPDF.Data))
		isWatermarked = true
	}

	_, err = h.r2.UploadObject(r.Context(), storage.UploadObjectParams{
		Key:         storageKey,
		Body:        uploads.Reader(fileData),
		ContentType: validatedFile.ContentType,
		SizeBytes:   fileSizeBytes,
	})
	if err != nil {
		slog.Error("failed to upload note file to r2", "error", err)
		h.renderIndexWithError(w, r, "Failed to upload file.")
		return
	}

	_, err = h.noteRepo.Create(r.Context(), academic.CreateNoteParams{
		ChapterID:        chapterID,
		Title:            title,
		Description:      description,
		OriginalFileName: validatedFile.OriginalFileName,
		StoredFileName:   storedFileName,
		StorageKey:       storageKey,
		ContentType:      validatedFile.ContentType,
		FileSizeBytes:    fileSizeBytes,
		IsPDF:            validatedFile.IsPDF,
		IsWatermarked:    isWatermarked,
		SortOrder:        sortOrder,
		IsPublished:      isPublished,
		UploadedBy:       uploadedBy,
	})
	if err != nil {
		if isUniqueViolation(err) {
			h.renderIndexWithError(w, r, "A note with this title already exists for the selected chapter.")
			return
		}

		slog.Error("failed to save note metadata", "error", err)
		h.renderIndexWithError(w, r, "File uploaded, but failed to save note metadata.")
		return
	}

	http.Redirect(w, r, "/admin/notes", http.StatusSeeOther)
}

func (h *AdminNoteHandler) renderIndexWithError(w http.ResponseWriter, r *http.Request, message string) {
	pageData, err := h.pageData(r)
	if err != nil {
		slog.Error("failed to load notes page data after error", "error", err)
		http.Error(w, "Failed to load notes", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusUnprocessableEntity)

	h.renderer.Render(w, r, "admin_notes.tmpl", views.TemplateData{
		Title: "Notes",
		Error: message,
		Data:  pageData,
	})
}

func (h *AdminNoteHandler) pageData(r *http.Request) (AdminNotesPageData, error) {
	classes, err := h.classRepo.List(r.Context())
	if err != nil {
		return AdminNotesPageData{}, err
	}

	notes, err := h.noteRepo.List(r.Context())
	if err != nil {
		return AdminNotesPageData{}, err
	}

	noteItems := make([]AdminNoteListItem, 0, len(notes))

	for _, note := range notes {
		fileURL, err := h.fileProxySigner.SignedFileURL(note.StorageKey)
		if err != nil {
			return AdminNotesPageData{}, fmt.Errorf("sign note file url: %w", err)
		}

		noteItems = append(noteItems, AdminNoteListItem{
			Note:    note,
			FileURL: fileURL,
		})
	}

	return AdminNotesPageData{
		Classes: classes,
		Notes:   noteItems,
	}, nil
}

func buildStoredFileName(noteID uuid.UUID, originalFileName string) string {
	extension := strings.ToLower(filepath.Ext(originalFileName))
	if extension == "" {
		extension = ".bin"
	}

	return fmt.Sprintf("%s%s", noteID.String(), extension)
}

func buildNoteStorageKey(chapterID uuid.UUID, noteID uuid.UUID, storedFileName string) string {
	now := time.Now().UTC()

	return fmt.Sprintf(
		"notes/%d/%02d/%s/%s/%s",
		now.Year(),
		now.Month(),
		chapterID.String(),
		noteID.String(),
		storedFileName,
	)
}

func (h *AdminNoteHandler) Edit(w http.ResponseWriter, r *http.Request) {
	noteID, err := uuid.Parse(chi.URLParam(r, "noteID"))
	if err != nil {
		http.NotFound(w, r)
		return
	}

	pageData, err := h.editPageData(r, noteID)
	if err != nil {
		if errors.Is(err, academic.ErrNoteNotFound) {
			http.NotFound(w, r)
			return
		}

		slog.Error("failed to load note edit page", "error", err)
		http.Error(w, "Failed to load note", http.StatusInternalServerError)
		return
	}

	h.renderer.Render(w, r, "admin_note_edit.tmpl", views.TemplateData{
		Title: "Edit Note",
		Data:  pageData,
	})
}

func (h *AdminNoteHandler) Update(w http.ResponseWriter, r *http.Request) {
	noteID, err := uuid.Parse(chi.URLParam(r, "noteID"))
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if err := r.ParseForm(); err != nil {
		h.renderEditWithError(w, r, noteID, "Invalid form submission.")
		return
	}

	chapterIDValue := strings.TrimSpace(r.PostForm.Get("chapter_id"))
	title := strings.TrimSpace(r.PostForm.Get("title"))
	description := strings.TrimSpace(r.PostForm.Get("description"))
	sortOrderValue := strings.TrimSpace(r.PostForm.Get("sort_order"))
	isPublished := r.PostForm.Get("is_published") == "on"

	chapterID, err := uuid.Parse(chapterIDValue)
	if err != nil {
		h.renderEditWithError(w, r, noteID, "Please select a valid chapter.")
		return
	}

	sortOrder := 0
	if sortOrderValue != "" {
		parsedSortOrder, err := strconv.Atoi(sortOrderValue)
		if err != nil {
			h.renderEditWithError(w, r, noteID, "Sort order must be a number.")
			return
		}

		sortOrder = parsedSortOrder
	}

	if title == "" {
		h.renderEditWithError(w, r, noteID, "Note title is required.")
		return
	}

	_, err = h.noteRepo.UpdateMetadata(r.Context(), academic.UpdateNoteMetadataParams{
		ID:          noteID,
		ChapterID:   chapterID,
		Title:       title,
		Description: description,
		SortOrder:   sortOrder,
		IsPublished: isPublished,
	})
	if err != nil {
		if isUniqueViolation(err) {
			h.renderEditWithError(w, r, noteID, "A note with this title already exists for the selected chapter.")
			return
		}

		if errors.Is(err, academic.ErrNoteNotFound) {
			http.NotFound(w, r)
			return
		}

		slog.Error("failed to update note metadata", "error", err)
		h.renderEditWithError(w, r, noteID, "Failed to update note.")
		return
	}

	http.Redirect(w, r, "/admin/notes", http.StatusSeeOther)
}

func (h *AdminNoteHandler) renderEditWithError(w http.ResponseWriter, r *http.Request, noteID uuid.UUID, message string) {
	pageData, err := h.editPageData(r, noteID)
	if err != nil {
		slog.Error("failed to reload note edit page after error", "error", err)
		http.Error(w, "Failed to load note", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusUnprocessableEntity)

	h.renderer.Render(w, r, "admin_note_edit.tmpl", views.TemplateData{
		Title: "Edit Note",
		Error: message,
		Data:  pageData,
	})
}

func (h *AdminNoteHandler) editPageData(r *http.Request, noteID uuid.UUID) (AdminNoteEditPageData, error) {
	chapters, err := h.chapterRepo.List(r.Context())
	if err != nil {
		return AdminNoteEditPageData{}, err
	}

	noteItem, err := h.noteRepo.FindByID(r.Context(), noteID)
	if err != nil {
		return AdminNoteEditPageData{}, err
	}

	fileURL, err := h.fileProxySigner.SignedFileURL(noteItem.StorageKey)
	if err != nil {
		return AdminNoteEditPageData{}, fmt.Errorf("sign note file url: %w", err)
	}

	return AdminNoteEditPageData{
		Chapters: chapters,
		Note:     noteItem,
		FileURL:  fileURL,
	}, nil
}

func (h *AdminNoteHandler) Archive(w http.ResponseWriter, r *http.Request) {
	noteID, err := uuid.Parse(chi.URLParam(r, "noteID"))
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if err := h.noteRepo.Archive(r.Context(), noteID); err != nil {
		if errors.Is(err, academic.ErrNoteNotFound) {
			http.NotFound(w, r)
			return
		}

		slog.Error("failed to archive note", "error", err)
		http.Error(w, "Failed to archive note", http.StatusInternalServerError)
		return
	}

	h.sessionManager.Put(r.Context(), "flash", "Note archived successfully.")
	http.Redirect(w, r, "/admin/notes", http.StatusSeeOther)
}

func (h *AdminNoteHandler) Unarchive(w http.ResponseWriter, r *http.Request) {
	noteID, err := uuid.Parse(chi.URLParam(r, "noteID"))
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if err := h.noteRepo.Unarchive(r.Context(), noteID); err != nil {
		if errors.Is(err, academic.ErrNoteNotFound) {
			http.NotFound(w, r)
			return
		}

		slog.Error("failed to unarchive note", "error", err)
		http.Error(w, "Failed to unarchive note", http.StatusInternalServerError)
		return
	}

	h.sessionManager.Put(r.Context(), "flash", "Note restored successfully.")
	http.Redirect(w, r, "/admin/notes", http.StatusSeeOther)
}
