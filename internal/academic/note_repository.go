package academic

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNoteNotFound = errors.New("note not found")

type Note struct {
	ID               uuid.UUID
	ChapterID        uuid.UUID
	ChapterName      string
	UnitName         string
	SubjectName      string
	SemesterName     string
	ClassName        string
	Title            string
	Slug             string
	Description      *string
	OriginalFileName string
	StoredFileName   string
	StorageKey       string
	ContentType      string
	FileSizeBytes    int64
	IsPDF            bool
	IsWatermarked    bool
	DownloadCount    int64
	ViewCount        int64
	SortOrder        int
	IsPublished      bool
	UploadedBy       *uuid.UUID
	ArchivedAt       *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type NoteRepository struct {
	pool *pgxpool.Pool
}

func NewNoteRepository(pool *pgxpool.Pool) *NoteRepository {
	return &NoteRepository{
		pool: pool,
	}
}

type NoteListFilter string

const (
	NoteListFilterAll      NoteListFilter = "all"
	NoteListFilterActive   NoteListFilter = "active"
	NoteListFilterArchived NoteListFilter = "archived"
)

type ListNotesParams struct {
	Search string
	Filter NoteListFilter
	Limit  int
	Offset int
}

type PaginatedNotes struct {
	Notes      []Note
	TotalCount int
}

type CreateNoteParams struct {
	ChapterID        uuid.UUID
	Title            string
	Description      string
	OriginalFileName string
	StoredFileName   string
	StorageKey       string
	ContentType      string
	FileSizeBytes    int64
	IsPDF            bool
	IsWatermarked    bool
	SortOrder        int
	IsPublished      bool
	UploadedBy       uuid.UUID
}

type UpdateNoteFileParams struct {
	ID               uuid.UUID
	OriginalFileName string
	StoredFileName   string
	StorageKey       string
	ContentType      string
	FileSizeBytes    int64
	IsPDF            bool
	IsWatermarked    bool
}

func (r *NoteRepository) Create(ctx context.Context, params CreateNoteParams) (Note, error) {
	title := strings.TrimSpace(params.Title)
	description := strings.TrimSpace(params.Description)
	originalFileName := strings.TrimSpace(params.OriginalFileName)
	storedFileName := strings.TrimSpace(params.StoredFileName)
	storageKey := strings.TrimSpace(params.StorageKey)
	contentType := strings.TrimSpace(params.ContentType)

	if params.ChapterID == uuid.Nil {
		return Note{}, errors.New("chapter is required")
	}

	if title == "" {
		return Note{}, errors.New("note title is required")
	}

	if originalFileName == "" {
		return Note{}, errors.New("original file name is required")
	}

	if storedFileName == "" {
		return Note{}, errors.New("stored file name is required")
	}

	if storageKey == "" {
		return Note{}, errors.New("storage key is required")
	}

	if contentType == "" {
		return Note{}, errors.New("content type is required")
	}

	if params.FileSizeBytes <= 0 {
		return Note{}, errors.New("file size must be greater than zero")
	}

	if params.UploadedBy == uuid.Nil {
		return Note{}, errors.New("uploaded by is required")
	}

	slug := Slugify(title)
	if slug == "" {
		return Note{}, errors.New("note title must contain valid characters")
	}

	var descriptionPtr *string
	if description != "" {
		descriptionPtr = &description
	}

	var created Note

	err := r.pool.QueryRow(ctx, `
		INSERT INTO notes (
			chapter_id,
			title,
			slug,
			description,
			original_file_name,
			stored_file_name,
			storage_key,
			content_type,
			file_size_bytes,
			is_pdf,
			is_watermarked,
			sort_order,
			is_published,
			uploaded_by
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING
			id,
			chapter_id,
			title,
			slug,
			description,
			original_file_name,
			stored_file_name,
			storage_key,
			content_type,
			file_size_bytes,
			is_pdf,
			is_watermarked,
			download_count,
			view_count,
			sort_order,
			is_published,
			uploaded_by,
			archived_at,
			created_at,
			updated_at
	`,
		params.ChapterID,
		title,
		slug,
		descriptionPtr,
		originalFileName,
		storedFileName,
		storageKey,
		contentType,
		params.FileSizeBytes,
		params.IsPDF,
		params.IsWatermarked,
		params.SortOrder,
		params.IsPublished,
		params.UploadedBy,
	).Scan(
		&created.ID,
		&created.ChapterID,
		&created.Title,
		&created.Slug,
		&created.Description,
		&created.OriginalFileName,
		&created.StoredFileName,
		&created.StorageKey,
		&created.ContentType,
		&created.FileSizeBytes,
		&created.IsPDF,
		&created.IsWatermarked,
		&created.DownloadCount,
		&created.ViewCount,
		&created.SortOrder,
		&created.IsPublished,
		&created.UploadedBy,
		&created.ArchivedAt,
		&created.CreatedAt,
		&created.UpdatedAt,
	)
	if err != nil {
		return Note{}, fmt.Errorf("create note: %w", err)
	}

	return created, nil
}

func (r *NoteRepository) List(ctx context.Context) ([]Note, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT
			n.id,
			n.chapter_id,
			ch.name AS chapter_name,
			u.name AS unit_name,
			sub.name AS subject_name,
			sem.name AS semester_name,
			c.name AS class_name,
			n.title,
			n.slug,
			n.description,
			n.original_file_name,
			n.stored_file_name,
			n.storage_key,
			n.content_type,
			n.file_size_bytes,
			n.is_pdf,
			n.is_watermarked,
			n.download_count,
			n.view_count,
			n.sort_order,
			n.is_published,
			n.uploaded_by,
			n.archived_at,
			n.created_at,
			n.updated_at
		FROM notes n
		JOIN chapters ch ON ch.id = n.chapter_id
		JOIN units u ON u.id = ch.unit_id
		JOIN subjects sub ON sub.id = u.subject_id
		JOIN semesters sem ON sem.id = sub.semester_id
		JOIN classes c ON c.id = sem.class_id
		ORDER BY
			n.archived_at IS NOT NULL ASC,
			c.sort_order ASC,
			c.name ASC,
			sem.sort_order ASC,
			sem.name ASC,
			sub.sort_order ASC,
			sub.name ASC,
			u.sort_order ASC,
			u.name ASC,
			ch.sort_order ASC,
			ch.name ASC,
			n.sort_order ASC,
			n.title ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list notes: %w", err)
	}
	defer rows.Close()

	notes := make([]Note, 0)

	for rows.Next() {
		var item Note

		err := rows.Scan(
			&item.ID,
			&item.ChapterID,
			&item.ChapterName,
			&item.UnitName,
			&item.SubjectName,
			&item.SemesterName,
			&item.ClassName,
			&item.Title,
			&item.Slug,
			&item.Description,
			&item.OriginalFileName,
			&item.StoredFileName,
			&item.StorageKey,
			&item.ContentType,
			&item.FileSizeBytes,
			&item.IsPDF,
			&item.IsWatermarked,
			&item.DownloadCount,
			&item.ViewCount,
			&item.SortOrder,
			&item.IsPublished,
			&item.UploadedBy,
			&item.ArchivedAt,
			&item.CreatedAt,
			&item.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan note: %w", err)
		}

		notes = append(notes, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate notes: %w", err)
	}

	return notes, nil
}

func (r *NoteRepository) FindByID(ctx context.Context, id uuid.UUID) (Note, error) {
	if id == uuid.Nil {
		return Note{}, ErrNoteNotFound
	}

	var item Note

	err := r.pool.QueryRow(ctx, `
		SELECT
			n.id,
			n.chapter_id,
			ch.name AS chapter_name,
			u.name AS unit_name,
			sub.name AS subject_name,
			sem.name AS semester_name,
			c.name AS class_name,
			n.title,
			n.slug,
			n.description,
			n.original_file_name,
			n.stored_file_name,
			n.storage_key,
			n.content_type,
			n.file_size_bytes,
			n.is_pdf,
			n.is_watermarked,
			n.download_count,
			n.view_count,
			n.sort_order,
			n.is_published,
			n.uploaded_by,
			n.archived_at,
			n.created_at,
			n.updated_at
		FROM notes n
		JOIN chapters ch ON ch.id = n.chapter_id
		JOIN units u ON u.id = ch.unit_id
		JOIN subjects sub ON sub.id = u.subject_id
		JOIN semesters sem ON sem.id = sub.semester_id
		JOIN classes c ON c.id = sem.class_id
		WHERE n.id = $1
	`, id).Scan(
		&item.ID,
		&item.ChapterID,
		&item.ChapterName,
		&item.UnitName,
		&item.SubjectName,
		&item.SemesterName,
		&item.ClassName,
		&item.Title,
		&item.Slug,
		&item.Description,
		&item.OriginalFileName,
		&item.StoredFileName,
		&item.StorageKey,
		&item.ContentType,
		&item.FileSizeBytes,
		&item.IsPDF,
		&item.IsWatermarked,
		&item.DownloadCount,
		&item.ViewCount,
		&item.SortOrder,
		&item.IsPublished,
		&item.UploadedBy,
		&item.ArchivedAt,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Note{}, ErrNoteNotFound
		}

		return Note{}, fmt.Errorf("find note by id: %w", err)
	}

	return item, nil
}

type UpdateNoteMetadataParams struct {
	ID          uuid.UUID
	ChapterID   uuid.UUID
	Title       string
	Description string
	SortOrder   int
	IsPublished bool
}

func (r *NoteRepository) UpdateMetadata(ctx context.Context, params UpdateNoteMetadataParams) (Note, error) {
	title := strings.TrimSpace(params.Title)
	description := strings.TrimSpace(params.Description)

	if params.ID == uuid.Nil {
		return Note{}, ErrNoteNotFound
	}

	if params.ChapterID == uuid.Nil {
		return Note{}, errors.New("chapter is required")
	}

	if title == "" {
		return Note{}, errors.New("note title is required")
	}

	slug := Slugify(title)
	if slug == "" {
		return Note{}, errors.New("note title must contain valid characters")
	}

	var descriptionPtr *string
	if description != "" {
		descriptionPtr = &description
	}

	var updated Note

	err := r.pool.QueryRow(ctx, `
		UPDATE notes
		SET
			chapter_id = $2,
			title = $3,
			slug = $4,
			description = $5,
			sort_order = $6,
			is_published = $7
		WHERE id = $1
		RETURNING
			id,
			chapter_id,
			title,
			slug,
			description,
			original_file_name,
			stored_file_name,
			storage_key,
			content_type,
			file_size_bytes,
			is_pdf,
			is_watermarked,
			download_count,
			view_count,
			sort_order,
			is_published,
			uploaded_by,
			archived_at,
			created_at,
			updated_at
	`,
		params.ID,
		params.ChapterID,
		title,
		slug,
		descriptionPtr,
		params.SortOrder,
		params.IsPublished,
	).Scan(
		&updated.ID,
		&updated.ChapterID,
		&updated.Title,
		&updated.Slug,
		&updated.Description,
		&updated.OriginalFileName,
		&updated.StoredFileName,
		&updated.StorageKey,
		&updated.ContentType,
		&updated.FileSizeBytes,
		&updated.IsPDF,
		&updated.IsWatermarked,
		&updated.DownloadCount,
		&updated.ViewCount,
		&updated.SortOrder,
		&updated.IsPublished,
		&updated.UploadedBy,
		&updated.ArchivedAt,
		&updated.CreatedAt,
		&updated.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Note{}, ErrNoteNotFound
		}

		return Note{}, fmt.Errorf("update note metadata: %w", err)
	}

	return updated, nil
}

func (r *NoteRepository) Archive(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return ErrNoteNotFound
	}

	commandTag, err := r.pool.Exec(ctx, `
		UPDATE notes
		SET
			archived_at = now(),
			is_published = FALSE
		WHERE id = $1
		AND archived_at IS NULL
	`, id)
	if err != nil {
		return fmt.Errorf("archive note: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return ErrNoteNotFound
	}

	return nil
}

func (r *NoteRepository) Unarchive(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return ErrNoteNotFound
	}

	commandTag, err := r.pool.Exec(ctx, `
		UPDATE notes
		SET archived_at = NULL
		WHERE id = $1
		AND archived_at IS NOT NULL
	`, id)
	if err != nil {
		return fmt.Errorf("unarchive note: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return ErrNoteNotFound
	}

	return nil
}

func (r *NoteRepository) ListPaginated(ctx context.Context, params ListNotesParams) (PaginatedNotes, error) {
	search := strings.TrimSpace(params.Search)
	filter := params.Filter

	if filter == "" {
		filter = NoteListFilterActive
	}

	limit := params.Limit
	if limit <= 0 {
		limit = 20
	}

	if limit > 100 {
		limit = 100
	}

	offset := params.Offset
	if offset < 0 {
		offset = 0
	}

	where := `
		WHERE (
			$1 = ''
			OR n.title ILIKE '%' || $1 || '%'
			OR n.original_file_name ILIKE '%' || $1 || '%'
			OR ch.name ILIKE '%' || $1 || '%'
			OR u.name ILIKE '%' || $1 || '%'
			OR sub.name ILIKE '%' || $1 || '%'
			OR sem.name ILIKE '%' || $1 || '%'
			OR c.name ILIKE '%' || $1 || '%'
		)
		AND (
			$2 = 'all'
			OR ($2 = 'active' AND n.archived_at IS NULL)
			OR ($2 = 'archived' AND n.archived_at IS NOT NULL)
		)
	`

	var totalCount int

	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM notes n
		JOIN chapters ch ON ch.id = n.chapter_id
		JOIN units u ON u.id = ch.unit_id
		JOIN subjects sub ON sub.id = u.subject_id
		JOIN semesters sem ON sem.id = sub.semester_id
		JOIN classes c ON c.id = sem.class_id
	`+where, search, string(filter)).Scan(&totalCount)
	if err != nil {
		return PaginatedNotes{}, fmt.Errorf("count notes: %w", err)
	}

	rows, err := r.pool.Query(ctx, `
		SELECT
			n.id,
			n.chapter_id,
			ch.name AS chapter_name,
			u.name AS unit_name,
			sub.name AS subject_name,
			sem.name AS semester_name,
			c.name AS class_name,
			n.title,
			n.slug,
			n.description,
			n.original_file_name,
			n.stored_file_name,
			n.storage_key,
			n.content_type,
			n.file_size_bytes,
			n.is_pdf,
			n.is_watermarked,
			n.download_count,
			n.view_count,
			n.sort_order,
			n.is_published,
			n.uploaded_by,
			n.archived_at,
			n.created_at,
			n.updated_at
		FROM notes n
		JOIN chapters ch ON ch.id = n.chapter_id
		JOIN units u ON u.id = ch.unit_id
		JOIN subjects sub ON sub.id = u.subject_id
		JOIN semesters sem ON sem.id = sub.semester_id
		JOIN classes c ON c.id = sem.class_id
	`+where+`
		ORDER BY
			n.archived_at IS NOT NULL ASC,
			n.created_at DESC,
			n.title ASC
		LIMIT $3 OFFSET $4
	`, search, string(filter), limit, offset)
	if err != nil {
		return PaginatedNotes{}, fmt.Errorf("list paginated notes: %w", err)
	}
	defer rows.Close()

	notes := make([]Note, 0)

	for rows.Next() {
		var item Note

		err := rows.Scan(
			&item.ID,
			&item.ChapterID,
			&item.ChapterName,
			&item.UnitName,
			&item.SubjectName,
			&item.SemesterName,
			&item.ClassName,
			&item.Title,
			&item.Slug,
			&item.Description,
			&item.OriginalFileName,
			&item.StoredFileName,
			&item.StorageKey,
			&item.ContentType,
			&item.FileSizeBytes,
			&item.IsPDF,
			&item.IsWatermarked,
			&item.DownloadCount,
			&item.ViewCount,
			&item.SortOrder,
			&item.IsPublished,
			&item.UploadedBy,
			&item.ArchivedAt,
			&item.CreatedAt,
			&item.UpdatedAt,
		)
		if err != nil {
			return PaginatedNotes{}, fmt.Errorf("scan paginated note: %w", err)
		}

		notes = append(notes, item)
	}

	if err := rows.Err(); err != nil {
		return PaginatedNotes{}, fmt.Errorf("iterate paginated notes: %w", err)
	}

	return PaginatedNotes{
		Notes:      notes,
		TotalCount: totalCount,
	}, nil
}

func (r *NoteRepository) DeleteArchived(ctx context.Context, id uuid.UUID) (Note, error) {
	if id == uuid.Nil {
		return Note{}, ErrNoteNotFound
	}

	var deleted Note

	err := r.pool.QueryRow(ctx, `
		DELETE FROM notes
		WHERE id = $1
		AND archived_at IS NOT NULL
		RETURNING
			id,
			chapter_id,
			title,
			slug,
			description,
			original_file_name,
			stored_file_name,
			storage_key,
			content_type,
			file_size_bytes,
			is_pdf,
			is_watermarked,
			download_count,
			view_count,
			sort_order,
			is_published,
			uploaded_by,
			archived_at,
			created_at,
			updated_at
	`, id).Scan(
		&deleted.ID,
		&deleted.ChapterID,
		&deleted.Title,
		&deleted.Slug,
		&deleted.Description,
		&deleted.OriginalFileName,
		&deleted.StoredFileName,
		&deleted.StorageKey,
		&deleted.ContentType,
		&deleted.FileSizeBytes,
		&deleted.IsPDF,
		&deleted.IsWatermarked,
		&deleted.DownloadCount,
		&deleted.ViewCount,
		&deleted.SortOrder,
		&deleted.IsPublished,
		&deleted.UploadedBy,
		&deleted.ArchivedAt,
		&deleted.CreatedAt,
		&deleted.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Note{}, ErrNoteNotFound
		}

		return Note{}, fmt.Errorf("delete archived note: %w", err)
	}

	return deleted, nil
}

func (r *NoteRepository) UpdateFile(ctx context.Context, params UpdateNoteFileParams) (Note, error) {
	if params.ID == uuid.Nil {
		return Note{}, ErrNoteNotFound
	}

	originalFileName := strings.TrimSpace(params.OriginalFileName)
	storedFileName := strings.TrimSpace(params.StoredFileName)
	storageKey := strings.TrimSpace(params.StorageKey)
	contentType := strings.TrimSpace(params.ContentType)

	if originalFileName == "" {
		return Note{}, errors.New("original file name is required")
	}

	if storedFileName == "" {
		return Note{}, errors.New("stored file name is required")
	}

	if storageKey == "" {
		return Note{}, errors.New("storage key is required")
	}

	if contentType == "" {
		return Note{}, errors.New("content type is required")
	}

	if params.FileSizeBytes <= 0 {
		return Note{}, errors.New("file size must be greater than zero")
	}

	var updated Note

	err := r.pool.QueryRow(ctx, `
		UPDATE notes
		SET
			original_file_name = $2,
			stored_file_name = $3,
			storage_key = $4,
			content_type = $5,
			file_size_bytes = $6,
			is_pdf = $7,
			is_watermarked = $8
		WHERE id = $1
		RETURNING
			id,
			chapter_id,
			title,
			slug,
			description,
			original_file_name,
			stored_file_name,
			storage_key,
			content_type,
			file_size_bytes,
			is_pdf,
			is_watermarked,
			download_count,
			view_count,
			sort_order,
			is_published,
			uploaded_by,
			archived_at,
			created_at,
			updated_at
	`,
		params.ID,
		originalFileName,
		storedFileName,
		storageKey,
		contentType,
		params.FileSizeBytes,
		params.IsPDF,
		params.IsWatermarked,
	).Scan(
		&updated.ID,
		&updated.ChapterID,
		&updated.Title,
		&updated.Slug,
		&updated.Description,
		&updated.OriginalFileName,
		&updated.StoredFileName,
		&updated.StorageKey,
		&updated.ContentType,
		&updated.FileSizeBytes,
		&updated.IsPDF,
		&updated.IsWatermarked,
		&updated.DownloadCount,
		&updated.ViewCount,
		&updated.SortOrder,
		&updated.IsPublished,
		&updated.UploadedBy,
		&updated.ArchivedAt,
		&updated.CreatedAt,
		&updated.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Note{}, ErrNoteNotFound
		}

		return Note{}, fmt.Errorf("update note file: %w", err)
	}

	return updated, nil
}

func (r *NoteRepository) IncrementViewCount(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return ErrNoteNotFound
	}

	commandTag, err := r.pool.Exec(ctx, `
		UPDATE notes
		SET view_count = view_count + 1
		WHERE id = $1
	`, id)
	if err != nil {
		return fmt.Errorf("increment note view count: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return ErrNoteNotFound
	}

	return nil
}
