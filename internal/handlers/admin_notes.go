package handlers

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/ifaisalabid1/notes-platform/internal/academic"
	"github.com/ifaisalabid1/notes-platform/internal/audit"
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
	auditRepo       *audit.Repository
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
	auditRepo *audit.Repository,
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
		auditRepo:       auditRepo,
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
	Classes    []academic.Class
	Notes      []AdminNoteListItem
	Pagination AdminNotesPagination
}

type AdminNotesPagination struct {
	Search      string
	Filter      string
	Page        int
	PerPage     int
	TotalCount  int
	TotalPages  int
	HasPrevious bool
	HasNext     bool
	PreviousURL string
	NextURL     string
}

type AdminNoteEditPageData struct {
	Chapters []academic.Chapter
	Note     academic.Note
	FileURL  string
}

type AdminNoteDeletePageData struct {
	Note    academic.Note
	FileURL string
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
	r.Body = http.MaxBytesReader(w, r.Body, uploads.MaxUploadSizeBytes)

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		h.renderIndexWithError(w, r, "Invalid upload. File may be too large.")
		return
	}

	defer func() {
		if r.MultipartForm != nil {
			_ = r.MultipartForm.RemoveAll()
		}
	}()

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

	if validatedFile.IsZIP {
		imageFiles, err := uploads.ValidateZipImageFiles(validatedFile)
		if err != nil {
			h.renderIndexWithError(w, r, err.Error())
			return
		}

		createdNotes, err := h.storeZipImageNotes(r, chapterID, title, description, sortOrder, isPublished, uploadedBy, imageFiles)
		if err != nil {
			slog.Error("failed to import zip image notes", "error", err)
			h.renderIndexWithError(w, r, err.Error())
			return
		}

		writeAuditLog(
			r,
			h.sessionManager,
			h.auditRepo,
			"note_zip_imported",
			"note",
			nil,
			"Imported gallery images from ZIP",
			map[string]any{
				"chapter_id":     chapterID.String(),
				"image_count":    len(createdNotes),
				"is_published":   isPublished,
				"zip_file_name":  validatedFile.OriginalFileName,
				"title_prefix":   title,
				"first_note_id":  createdNotes[0].ID.String(),
				"first_note_key": createdNotes[0].StorageKey,
			},
		)

		h.sessionManager.Put(r.Context(), "flash", fmt.Sprintf("Imported %d gallery image%s.", len(createdNotes), pluralSuffix(len(createdNotes))))
		http.Redirect(w, r, "/admin/notes", http.StatusSeeOther)
		return
	}

	if title == "" {
		h.renderIndexWithError(w, r, "Note title is required.")
		return
	}

	createdNote, err := h.storeUploadedNote(r, storeUploadedNoteParams{
		ChapterID:      chapterID,
		Title:          title,
		Description:    description,
		SortOrder:      sortOrder,
		IsPublished:    isPublished,
		UploadedBy:     uploadedBy,
		File:           validatedFile,
		AllowTitleCopy: false,
	})
	if err != nil {
		h.renderIndexWithError(w, r, err.Error())
		return
	}

	writeAuditLog(
		r,
		h.sessionManager,
		h.auditRepo,
		"note_uploaded",
		"note",
		&createdNote.ID,
		"Uploaded note file",
		map[string]any{
			"title":              createdNote.Title,
			"original_file_name": createdNote.OriginalFileName,
			"storage_key":        createdNote.StorageKey,
			"content_type":       createdNote.ContentType,
			"file_size_bytes":    createdNote.FileSizeBytes,
			"is_watermarked":     createdNote.IsWatermarked,
			"is_published":       createdNote.IsPublished,
		},
	)

	http.Redirect(w, r, "/admin/notes", http.StatusSeeOther)
}

type storeUploadedNoteParams struct {
	ChapterID      uuid.UUID
	Title          string
	Description    string
	SortOrder      int
	IsPublished    bool
	UploadedBy     uuid.UUID
	File           uploads.ValidatedFile
	AllowTitleCopy bool
}

func (h *AdminNoteHandler) storeZipImageNotes(
	r *http.Request,
	chapterID uuid.UUID,
	titlePrefix string,
	description string,
	sortOrder int,
	isPublished bool,
	uploadedBy uuid.UUID,
	files []uploads.ValidatedFile,
) ([]academic.Note, error) {
	createdNotes := make([]academic.Note, 0, len(files))

	for index, file := range files {
		imageTitle := imageNoteTitle(titlePrefix, file.OriginalFileName, index+1)

		createdNote, err := h.storeUploadedNote(r, storeUploadedNoteParams{
			ChapterID:      chapterID,
			Title:          imageTitle,
			Description:    description,
			SortOrder:      sortOrder + index,
			IsPublished:    isPublished,
			UploadedBy:     uploadedBy,
			File:           file,
			AllowTitleCopy: true,
		})
		if err != nil {
			return createdNotes, fmt.Errorf("import %s: %w", file.OriginalFileName, err)
		}

		createdNotes = append(createdNotes, createdNote)
	}

	return createdNotes, nil
}

func (h *AdminNoteHandler) storeUploadedNote(r *http.Request, params storeUploadedNoteParams) (academic.Note, error) {
	noteID := uuid.New()
	storedFileName := buildStoredFileName(noteID, params.File.OriginalFileName)
	storageKey := buildNoteStorageKey(params.ChapterID, noteID, storedFileName)

	fileData := params.File.Data
	fileSizeBytes := params.File.SizeBytes
	isWatermarked := false

	if params.File.IsPDF {
		watermarkedPDF, err := h.pdfWatermarker.Apply(params.File.Data)
		if err != nil {
			slog.Error("failed to watermark pdf", "error", err)
			return academic.Note{}, errors.New("failed to watermark PDF")
		}

		fileData = watermarkedPDF.Data
		fileSizeBytes = int64(len(watermarkedPDF.Data))
		isWatermarked = true
	}

	_, err := h.r2.UploadObject(r.Context(), storage.UploadObjectParams{
		Key:         storageKey,
		Body:        uploads.Reader(fileData),
		ContentType: params.File.ContentType,
		SizeBytes:   fileSizeBytes,
	})
	if err != nil {
		slog.Error("failed to upload note file to r2", "error", err)
		return academic.Note{}, errors.New("failed to upload file")
	}

	createParams := academic.CreateNoteParams{
		ChapterID:        params.ChapterID,
		Title:            params.Title,
		Description:      params.Description,
		OriginalFileName: params.File.OriginalFileName,
		StoredFileName:   storedFileName,
		StorageKey:       storageKey,
		ContentType:      params.File.ContentType,
		FileSizeBytes:    fileSizeBytes,
		IsPDF:            params.File.IsPDF,
		IsWatermarked:    isWatermarked,
		SortOrder:        params.SortOrder,
		IsPublished:      params.IsPublished,
		UploadedBy:       params.UploadedBy,
	}

	createdNote, err := h.noteRepo.Create(r.Context(), createParams)
	if err != nil && params.AllowTitleCopy && isUniqueViolation(err) {
		createParams.Title = fmt.Sprintf("%s %s", params.Title, noteID.String()[:8])
		createdNote, err = h.noteRepo.Create(r.Context(), createParams)
	}
	if err != nil {
		if cleanupErr := h.r2.DeleteObject(r.Context(), storageKey); cleanupErr != nil {
			slog.Error(
				"failed to cleanup r2 object after note metadata error",
				"storage_key", storageKey,
				"error", cleanupErr,
			)
		}

		if isUniqueViolation(err) {
			return academic.Note{}, errors.New("a note with this title already exists for the selected chapter")
		}

		slog.Error("failed to save note metadata", "error", err)
		return academic.Note{}, errors.New("failed to save note metadata. Uploaded file was cleaned up")
	}

	return createdNote, nil
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

	query := r.URL.Query()

	search := strings.TrimSpace(query.Get("q"))
	filter := normalizeNoteFilter(query.Get("filter"))

	page := parsePositiveInt(query.Get("page"), 1)
	perPage := parsePositiveInt(query.Get("per_page"), 20)

	if perPage > 100 {
		perPage = 100
	}

	offset := (page - 1) * perPage

	paginatedNotes, err := h.noteRepo.ListPaginated(r.Context(), academic.ListNotesParams{
		Search: search,
		Filter: filter,
		Limit:  perPage,
		Offset: offset,
	})
	if err != nil {
		return AdminNotesPageData{}, err
	}

	totalPages := 0
	if paginatedNotes.TotalCount > 0 {
		totalPages = (paginatedNotes.TotalCount + perPage - 1) / perPage
	}

	if totalPages > 0 && page > totalPages {
		page = totalPages
		offset = (page - 1) * perPage

		paginatedNotes, err = h.noteRepo.ListPaginated(r.Context(), academic.ListNotesParams{
			Search: search,
			Filter: filter,
			Limit:  perPage,
			Offset: offset,
		})
		if err != nil {
			return AdminNotesPageData{}, err
		}
	}

	noteItems := make([]AdminNoteListItem, 0, len(paginatedNotes.Notes))

	for _, note := range paginatedNotes.Notes {
		fileURL, err := h.fileProxySigner.SignedFileURL(note.StorageKey)
		if err != nil {
			return AdminNotesPageData{}, fmt.Errorf("sign note file url: %w", err)
		}

		noteItems = append(noteItems, AdminNoteListItem{
			Note:    note,
			FileURL: fileURL,
		})
	}

	hasPrevious := page > 1
	hasNext := totalPages > 0 && page < totalPages

	previousURL := ""
	if hasPrevious {
		previousURL = buildAdminNotesURL(search, string(filter), page-1, perPage)
	}

	nextURL := ""
	if hasNext {
		nextURL = buildAdminNotesURL(search, string(filter), page+1, perPage)
	}

	return AdminNotesPageData{
		Classes: classes,
		Notes:   noteItems,
		Pagination: AdminNotesPagination{
			Search:      search,
			Filter:      string(filter),
			Page:        page,
			PerPage:     perPage,
			TotalCount:  paginatedNotes.TotalCount,
			TotalPages:  totalPages,
			HasPrevious: hasPrevious,
			HasNext:     hasNext,
			PreviousURL: previousURL,
			NextURL:     nextURL,
		},
	}, nil
}

func buildStoredFileName(noteID uuid.UUID, originalFileName string) string {
	extension := strings.ToLower(filepath.Ext(originalFileName))
	if extension == "" {
		extension = ".bin"
	}

	return fmt.Sprintf("%s%s", noteID.String(), extension)
}

func imageNoteTitle(prefix string, originalFileName string, index int) string {
	baseName := strings.TrimSpace(strings.TrimSuffix(filepath.Base(originalFileName), filepath.Ext(originalFileName)))
	baseName = strings.ReplaceAll(baseName, "_", " ")
	baseName = strings.ReplaceAll(baseName, "-", " ")
	baseName = strings.Join(strings.Fields(baseName), " ")

	if baseName == "" {
		baseName = fmt.Sprintf("Image %02d", index)
	}

	if strings.TrimSpace(prefix) == "" {
		return baseName
	}

	return fmt.Sprintf("%s - %s", strings.TrimSpace(prefix), baseName)
}

func pluralSuffix(count int) string {
	if count == 1 {
		return ""
	}

	return "s"
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

	updatedNote, err := h.noteRepo.UpdateMetadata(r.Context(), academic.UpdateNoteMetadataParams{
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

	writeAuditLog(
		r,
		h.sessionManager,
		h.auditRepo,
		"note_updated",
		"note",
		&updatedNote.ID,
		"Updated note metadata",
		map[string]any{
			"title":        updatedNote.Title,
			"chapter_id":   updatedNote.ChapterID.String(),
			"is_published": updatedNote.IsPublished,
			"sort_order":   updatedNote.SortOrder,
		},
	)

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

	writeAuditLog(
		r,
		h.sessionManager,
		h.auditRepo,
		"note_archived",
		"note",
		&noteID,
		"Archived note",
		map[string]any{},
	)

	h.sessionManager.Put(r.Context(), "flash", "Note archived successfully.")

	redirectURL := r.Header.Get("Referer")
	if redirectURL == "" {
		redirectURL = "/admin/notes"
	}

	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
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

	writeAuditLog(
		r,
		h.sessionManager,
		h.auditRepo,
		"note_restored",
		"note",
		&noteID,
		"Restored archived note",
		map[string]any{},
	)

	h.sessionManager.Put(r.Context(), "flash", "Note restored successfully.")

	redirectURL := r.Header.Get("Referer")
	if redirectURL == "" {
		redirectURL = "/admin/notes"
	}

	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

func parsePositiveInt(value string, fallback int) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || parsed <= 0 {
		return fallback
	}

	return parsed
}

func buildAdminNotesURL(search string, filter string, page int, perPage int) string {
	values := url.Values{}

	if strings.TrimSpace(search) != "" {
		values.Set("q", strings.TrimSpace(search))
	}

	if strings.TrimSpace(filter) != "" {
		values.Set("filter", strings.TrimSpace(filter))
	}

	values.Set("page", strconv.Itoa(page))
	values.Set("per_page", strconv.Itoa(perPage))

	return "/admin/notes?" + values.Encode()
}

func normalizeNoteFilter(value string) academic.NoteListFilter {
	switch academic.NoteListFilter(strings.TrimSpace(value)) {
	case academic.NoteListFilterAll:
		return academic.NoteListFilterAll
	case academic.NoteListFilterArchived:
		return academic.NoteListFilterArchived
	default:
		return academic.NoteListFilterActive
	}
}

func (h *AdminNoteHandler) ConfirmDeleteArchived(w http.ResponseWriter, r *http.Request) {
	noteID, err := uuid.Parse(chi.URLParam(r, "noteID"))
	if err != nil {
		http.NotFound(w, r)
		return
	}

	noteItem, err := h.noteRepo.FindByID(r.Context(), noteID)
	if err != nil {
		if errors.Is(err, academic.ErrNoteNotFound) {
			http.NotFound(w, r)
			return
		}

		slog.Error("failed to find note for delete confirmation", "error", err)
		http.Error(w, "Failed to load note", http.StatusInternalServerError)
		return
	}

	if noteItem.ArchivedAt == nil {
		h.sessionManager.Put(r.Context(), "flash", "Only archived notes can be permanently deleted.")
		http.Redirect(w, r, "/admin/notes", http.StatusSeeOther)
		return
	}

	fileURL, err := h.fileProxySigner.SignedFileURL(noteItem.StorageKey)
	if err != nil {
		slog.Error("failed to sign note file url for delete confirmation", "error", err)
		http.Error(w, "Failed to load note", http.StatusInternalServerError)
		return
	}

	h.renderer.Render(w, r, "admin_note_delete.tmpl", views.TemplateData{
		Title: "Delete Note",
		Data: AdminNoteDeletePageData{
			Note:    noteItem,
			FileURL: fileURL,
		},
	})
}

func (h *AdminNoteHandler) DeleteArchived(w http.ResponseWriter, r *http.Request) {
	noteID, err := uuid.Parse(chi.URLParam(r, "noteID"))
	if err != nil {
		http.NotFound(w, r)
		return
	}

	deletedNote, err := h.noteRepo.DeleteArchived(r.Context(), noteID)
	if err != nil {
		if errors.Is(err, academic.ErrNoteNotFound) {
			h.sessionManager.Put(r.Context(), "flash", "Only archived notes can be permanently deleted.")
			http.Redirect(w, r, "/admin/notes?filter=archived", http.StatusSeeOther)
			return
		}

		slog.Error("failed to permanently delete archived note", "error", err)
		http.Error(w, "Failed to delete note", http.StatusInternalServerError)
		return
	}

	if err := h.r2.DeleteObject(r.Context(), deletedNote.StorageKey); err != nil {
		slog.Error(
			"failed to delete r2 object after deleting note metadata",
			"note_id", deletedNote.ID.String(),
			"storage_key", deletedNote.StorageKey,
			"error", err,
		)

		h.sessionManager.Put(
			r.Context(),
			"flash",
			"Note deleted from database, but file cleanup failed. Check logs and R2 manually.",
		)
	} else {
		h.sessionManager.Put(r.Context(), "flash", "Archived note permanently deleted.")
	}

	writeAuditLog(
		r,
		h.sessionManager,
		h.auditRepo,
		"note_deleted",
		"note",
		&deletedNote.ID,
		"Permanently deleted archived note",
		map[string]any{
			"title":              deletedNote.Title,
			"original_file_name": deletedNote.OriginalFileName,
			"storage_key":        deletedNote.StorageKey,
		},
	)

	http.Redirect(w, r, "/admin/notes?filter=archived", http.StatusSeeOther)
}

func (h *AdminNoteHandler) ReplaceFile(w http.ResponseWriter, r *http.Request) {
	noteID, err := uuid.Parse(chi.URLParam(r, "noteID"))
	if err != nil {
		http.NotFound(w, r)
		return
	}

	existingNote, err := h.noteRepo.FindByID(r.Context(), noteID)
	if err != nil {
		if errors.Is(err, academic.ErrNoteNotFound) {
			http.NotFound(w, r)
			return
		}

		slog.Error("failed to find note before file replacement", "error", err)
		h.renderEditWithError(w, r, noteID, "Failed to load note.")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, uploads.MaxUploadSizeBytes)

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		h.renderEditWithError(w, r, noteID, "Invalid upload. File may be too large.")
		return
	}

	defer func() {
		if r.MultipartForm != nil {
			_ = r.MultipartForm.RemoveAll()
		}
	}()

	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		h.renderEditWithError(w, r, noteID, "Replacement file is required.")
		return
	}
	defer file.Close()

	validatedFile, err := uploads.ValidateUploadedFile(file, fileHeader)
	if err != nil {
		h.renderEditWithError(w, r, noteID, err.Error())
		return
	}

	storedFileName := fmt.Sprintf(
		"%s%s",
		uuid.NewString(),
		strings.ToLower(filepath.Ext(validatedFile.OriginalFileName)),
	)

	storageKey := fmt.Sprintf(
		"notes/%s/%s",
		time.Now().UTC().Format("2006/01/02"),
		storedFileName,
	)

	fileData := validatedFile.Data
	fileSizeBytes := validatedFile.SizeBytes
	isWatermarked := false

	if validatedFile.IsPDF {
		watermarkedPDF, err := h.pdfWatermarker.Apply(validatedFile.Data)
		if err != nil {
			slog.Error("failed to watermark replacement pdf", "error", err)
			h.renderEditWithError(w, r, noteID, "Failed to watermark replacement PDF.")
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
		slog.Error("failed to upload replacement file to r2", "error", err)
		h.renderEditWithError(w, r, noteID, "Failed to upload replacement file.")
		return
	}

	updatedNote, err := h.noteRepo.UpdateFile(r.Context(), academic.UpdateNoteFileParams{
		ID:               noteID,
		OriginalFileName: validatedFile.OriginalFileName,
		StoredFileName:   storedFileName,
		StorageKey:       storageKey,
		ContentType:      validatedFile.ContentType,
		FileSizeBytes:    fileSizeBytes,
		IsPDF:            validatedFile.IsPDF,
		IsWatermarked:    isWatermarked,
	})
	if err != nil {
		if cleanupErr := h.r2.DeleteObject(r.Context(), storageKey); cleanupErr != nil {
			slog.Error(
				"failed to cleanup replacement r2 object after note file update error",
				"storage_key", storageKey,
				"error", cleanupErr,
			)
		}

		if errors.Is(err, academic.ErrNoteNotFound) {
			http.NotFound(w, r)
			return
		}

		slog.Error("failed to update note file metadata", "error", err)
		h.renderEditWithError(w, r, noteID, "Failed to update note file metadata.")
		return
	}

	if err := h.r2.DeleteObject(r.Context(), existingNote.StorageKey); err != nil {
		slog.Error(
			"failed to delete old r2 object after file replacement",
			"note_id", noteID.String(),
			"old_storage_key", existingNote.StorageKey,
			"new_storage_key", storageKey,
			"error", err,
		)

		h.sessionManager.Put(
			r.Context(),
			"flash",
			"File replaced, but old file cleanup failed. Check logs and R2 manually.",
		)

		http.Redirect(w, r, "/admin/notes/"+noteID.String()+"/edit", http.StatusSeeOther)
		return
	}

	h.sessionManager.Put(r.Context(), "flash", "Note file replaced successfully.")

	writeAuditLog(
		r,
		h.sessionManager,
		h.auditRepo,
		"note_file_replaced",
		"note",
		&updatedNote.ID,
		"Replaced note file",
		map[string]any{
			"title":              updatedNote.Title,
			"old_storage_key":    existingNote.StorageKey,
			"new_storage_key":    updatedNote.StorageKey,
			"original_file_name": updatedNote.OriginalFileName,
			"content_type":       updatedNote.ContentType,
			"file_size_bytes":    updatedNote.FileSizeBytes,
			"is_watermarked":     updatedNote.IsWatermarked,
		},
	)

	http.Redirect(w, r, "/admin/notes/"+noteID.String()+"/edit", http.StatusSeeOther)
}
