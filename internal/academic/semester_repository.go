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

var ErrSemesterNotFound = errors.New("semester not found")

type Semester struct {
	ID          uuid.UUID
	ClassID     uuid.UUID
	ClassName   string
	Name        string
	Slug        string
	Description *string
	SortOrder   int
	IsPublished bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type SemesterRepository struct {
	pool *pgxpool.Pool
}

func NewSemesterRepository(pool *pgxpool.Pool) *SemesterRepository {
	return &SemesterRepository{
		pool: pool,
	}
}

type CreateSemesterParams struct {
	ClassID     uuid.UUID
	Name        string
	Description string
	SortOrder   int
	IsPublished bool
}

func (r *SemesterRepository) Create(ctx context.Context, params CreateSemesterParams) (Semester, error) {
	name := strings.TrimSpace(params.Name)
	description := strings.TrimSpace(params.Description)

	if params.ClassID == uuid.Nil {
		return Semester{}, errors.New("class is required")
	}

	if name == "" {
		return Semester{}, errors.New("semester name is required")
	}

	slug := Slugify(name)
	if slug == "" {
		return Semester{}, errors.New("semester name must contain valid characters")
	}

	var descriptionPtr *string
	if description != "" {
		descriptionPtr = &description
	}

	var created Semester

	err := r.pool.QueryRow(ctx, `
		INSERT INTO semesters (
			class_id,
			name,
			slug,
			description,
			sort_order,
			is_published
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING
			id,
			class_id,
			name,
			slug,
			description,
			sort_order,
			is_published,
			created_at,
			updated_at
	`, params.ClassID, name, slug, descriptionPtr, params.SortOrder, params.IsPublished).Scan(
		&created.ID,
		&created.ClassID,
		&created.Name,
		&created.Slug,
		&created.Description,
		&created.SortOrder,
		&created.IsPublished,
		&created.CreatedAt,
		&created.UpdatedAt,
	)
	if err != nil {
		return Semester{}, fmt.Errorf("create semester: %w", err)
	}

	return created, nil
}

func (r *SemesterRepository) List(ctx context.Context) ([]Semester, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT
			s.id,
			s.class_id,
			c.name AS class_name,
			s.name,
			s.slug,
			s.description,
			s.sort_order,
			s.is_published,
			s.created_at,
			s.updated_at
		FROM semesters s
		JOIN classes c ON c.id = s.class_id
		ORDER BY c.sort_order ASC, c.name ASC, s.sort_order ASC, s.name ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list semesters: %w", err)
	}
	defer rows.Close()

	semesters := make([]Semester, 0)

	for rows.Next() {
		var item Semester

		err := rows.Scan(
			&item.ID,
			&item.ClassID,
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
			return nil, fmt.Errorf("scan semester: %w", err)
		}

		semesters = append(semesters, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate semesters: %w", err)
	}

	return semesters, nil
}

func (r *SemesterRepository) FindByID(ctx context.Context, id uuid.UUID) (Semester, error) {
	if id == uuid.Nil {
		return Semester{}, ErrSemesterNotFound
	}

	var item Semester

	err := r.pool.QueryRow(ctx, `
		SELECT
			s.id,
			s.class_id,
			c.name AS class_name,
			s.name,
			s.slug,
			s.description,
			s.sort_order,
			s.is_published,
			s.created_at,
			s.updated_at
		FROM semesters s
		JOIN classes c ON c.id = s.class_id
		WHERE s.id = $1
	`, id).Scan(
		&item.ID,
		&item.ClassID,
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
			return Semester{}, ErrSemesterNotFound
		}

		return Semester{}, fmt.Errorf("find semester by id: %w", err)
	}

	return item, nil
}

type UpdateSemesterParams struct {
	ID          uuid.UUID
	ClassID     uuid.UUID
	Name        string
	Description string
	SortOrder   int
	IsPublished bool
}

func (r *SemesterRepository) Update(ctx context.Context, params UpdateSemesterParams) (Semester, error) {
	name := strings.TrimSpace(params.Name)
	description := strings.TrimSpace(params.Description)

	if params.ID == uuid.Nil {
		return Semester{}, ErrSemesterNotFound
	}

	if params.ClassID == uuid.Nil {
		return Semester{}, errors.New("class is required")
	}

	if name == "" {
		return Semester{}, errors.New("semester name is required")
	}

	slug := Slugify(name)
	if slug == "" {
		return Semester{}, errors.New("semester name must contain valid characters")
	}

	var descriptionPtr *string
	if description != "" {
		descriptionPtr = &description
	}

	var updated Semester

	err := r.pool.QueryRow(ctx, `
		UPDATE semesters
		SET
			class_id = $2,
			name = $3,
			slug = $4,
			description = $5,
			sort_order = $6,
			is_published = $7
		WHERE id = $1
		RETURNING
			id,
			class_id,
			name,
			slug,
			description,
			sort_order,
			is_published,
			created_at,
			updated_at
	`,
		params.ID,
		params.ClassID,
		name,
		slug,
		descriptionPtr,
		params.SortOrder,
		params.IsPublished,
	).Scan(
		&updated.ID,
		&updated.ClassID,
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
			return Semester{}, ErrSemesterNotFound
		}

		return Semester{}, fmt.Errorf("update semester: %w", err)
	}

	return updated, nil
}
