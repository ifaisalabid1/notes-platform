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

var ErrChapterNotFound = errors.New("chapter not found")

type Chapter struct {
	ID           uuid.UUID
	UnitID       uuid.UUID
	UnitName     string
	SubjectName  string
	SemesterName string
	ClassName    string
	Name         string
	Slug         string
	Description  *string
	SortOrder    int
	IsPublished  bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type ChapterRepository struct {
	pool *pgxpool.Pool
}

func NewChapterRepository(pool *pgxpool.Pool) *ChapterRepository {
	return &ChapterRepository{
		pool: pool,
	}
}

type CreateChapterParams struct {
	UnitID      uuid.UUID
	Name        string
	Description string
	SortOrder   int
	IsPublished bool
}

func (r *ChapterRepository) Create(ctx context.Context, params CreateChapterParams) (Chapter, error) {
	name := strings.TrimSpace(params.Name)
	description := strings.TrimSpace(params.Description)

	if params.UnitID == uuid.Nil {
		return Chapter{}, errors.New("unit is required")
	}

	if name == "" {
		return Chapter{}, errors.New("chapter name is required")
	}

	slug := Slugify(name)
	if slug == "" {
		return Chapter{}, errors.New("chapter name must contain valid characters")
	}

	var descriptionPtr *string
	if description != "" {
		descriptionPtr = &description
	}

	var created Chapter

	err := r.pool.QueryRow(ctx, `
		INSERT INTO chapters (
			unit_id,
			name,
			slug,
			description,
			sort_order,
			is_published
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING
			id,
			unit_id,
			name,
			slug,
			description,
			sort_order,
			is_published,
			created_at,
			updated_at
	`, params.UnitID, name, slug, descriptionPtr, params.SortOrder, params.IsPublished).Scan(
		&created.ID,
		&created.UnitID,
		&created.Name,
		&created.Slug,
		&created.Description,
		&created.SortOrder,
		&created.IsPublished,
		&created.CreatedAt,
		&created.UpdatedAt,
	)
	if err != nil {
		return Chapter{}, fmt.Errorf("create chapter: %w", err)
	}

	return created, nil
}

func (r *ChapterRepository) List(ctx context.Context) ([]Chapter, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT
			ch.id,
			ch.unit_id,
			u.name AS unit_name,
			sub.name AS subject_name,
			sem.name AS semester_name,
			c.name AS class_name,
			ch.name,
			ch.slug,
			ch.description,
			ch.sort_order,
			ch.is_published,
			ch.created_at,
			ch.updated_at
		FROM chapters ch
		JOIN units u ON u.id = ch.unit_id
		JOIN subjects sub ON sub.id = u.subject_id
		JOIN semesters sem ON sem.id = sub.semester_id
		JOIN classes c ON c.id = sem.class_id
		ORDER BY
			c.sort_order ASC,
			c.name ASC,
			sem.sort_order ASC,
			sem.name ASC,
			sub.sort_order ASC,
			sub.name ASC,
			u.sort_order ASC,
			u.name ASC,
			ch.sort_order ASC,
			ch.name ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list chapters: %w", err)
	}
	defer rows.Close()

	chapters := make([]Chapter, 0)

	for rows.Next() {
		var item Chapter

		err := rows.Scan(
			&item.ID,
			&item.UnitID,
			&item.UnitName,
			&item.SubjectName,
			&item.SemesterName,
			&item.ClassName,
			&item.Name,
			&item.Slug,
			&item.Description,
			&item.SortOrder,
			&item.IsPublished,
			&item.CreatedAt,
			&item.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan chapter: %w", err)
		}

		chapters = append(chapters, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate chapters: %w", err)
	}

	return chapters, nil
}

func (r *ChapterRepository) FindByID(ctx context.Context, id uuid.UUID) (Chapter, error) {
	if id == uuid.Nil {
		return Chapter{}, ErrChapterNotFound
	}

	var item Chapter

	err := r.pool.QueryRow(ctx, `
		SELECT
			ch.id,
			ch.unit_id,
			u.name AS unit_name,
			sub.name AS subject_name,
			sem.name AS semester_name,
			c.name AS class_name,
			ch.name,
			ch.slug,
			ch.description,
			ch.sort_order,
			ch.is_published,
			ch.created_at,
			ch.updated_at
		FROM chapters ch
		JOIN units u ON u.id = ch.unit_id
		JOIN subjects sub ON sub.id = u.subject_id
		JOIN semesters sem ON sem.id = sub.semester_id
		JOIN classes c ON c.id = sem.class_id
		WHERE ch.id = $1
	`, id).Scan(
		&item.ID,
		&item.UnitID,
		&item.UnitName,
		&item.SubjectName,
		&item.SemesterName,
		&item.ClassName,
		&item.Name,
		&item.Slug,
		&item.Description,
		&item.SortOrder,
		&item.IsPublished,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Chapter{}, ErrChapterNotFound
		}

		return Chapter{}, fmt.Errorf("find chapter by id: %w", err)
	}

	return item, nil
}

type UpdateChapterParams struct {
	ID          uuid.UUID
	UnitID      uuid.UUID
	Name        string
	Description string
	SortOrder   int
	IsPublished bool
}

func (r *ChapterRepository) Update(ctx context.Context, params UpdateChapterParams) (Chapter, error) {
	name := strings.TrimSpace(params.Name)
	description := strings.TrimSpace(params.Description)

	if params.ID == uuid.Nil {
		return Chapter{}, ErrChapterNotFound
	}

	if params.UnitID == uuid.Nil {
		return Chapter{}, errors.New("unit is required")
	}

	if name == "" {
		return Chapter{}, errors.New("chapter name is required")
	}

	slug := Slugify(name)
	if slug == "" {
		return Chapter{}, errors.New("chapter name must contain valid characters")
	}

	var descriptionPtr *string
	if description != "" {
		descriptionPtr = &description
	}

	var updated Chapter

	err := r.pool.QueryRow(ctx, `
		UPDATE chapters
		SET
			unit_id = $2,
			name = $3,
			slug = $4,
			description = $5,
			sort_order = $6,
			is_published = $7
		WHERE id = $1
		RETURNING
			id,
			unit_id,
			name,
			slug,
			description,
			sort_order,
			is_published,
			created_at,
			updated_at
	`,
		params.ID,
		params.UnitID,
		name,
		slug,
		descriptionPtr,
		params.SortOrder,
		params.IsPublished,
	).Scan(
		&updated.ID,
		&updated.UnitID,
		&updated.Name,
		&updated.Slug,
		&updated.Description,
		&updated.SortOrder,
		&updated.IsPublished,
		&updated.CreatedAt,
		&updated.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Chapter{}, ErrChapterNotFound
		}

		return Chapter{}, fmt.Errorf("update chapter: %w", err)
	}

	return updated, nil
}
