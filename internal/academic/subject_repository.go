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

var ErrSubjectNotFound = errors.New("subject not found")

type Subject struct {
	ID           uuid.UUID
	SemesterID   uuid.UUID
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

type SubjectRepository struct {
	pool *pgxpool.Pool
}

func NewSubjectRepository(pool *pgxpool.Pool) *SubjectRepository {
	return &SubjectRepository{
		pool: pool,
	}
}

type CreateSubjectParams struct {
	SemesterID  uuid.UUID
	Name        string
	Description string
	SortOrder   int
	IsPublished bool
}

func (r *SubjectRepository) Create(ctx context.Context, params CreateSubjectParams) (Subject, error) {
	name := strings.TrimSpace(params.Name)
	description := strings.TrimSpace(params.Description)

	if params.SemesterID == uuid.Nil {
		return Subject{}, errors.New("semester is required")
	}

	if name == "" {
		return Subject{}, errors.New("subject name is required")
	}

	slug := Slugify(name)
	if slug == "" {
		return Subject{}, errors.New("subject name must contain valid characters")
	}

	var descriptionPtr *string
	if description != "" {
		descriptionPtr = &description
	}

	var created Subject

	err := r.pool.QueryRow(ctx, `
		INSERT INTO subjects (
			semester_id,
			name,
			slug,
			description,
			sort_order,
			is_published
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING
			id,
			semester_id,
			name,
			slug,
			description,
			sort_order,
			is_published,
			created_at,
			updated_at
	`, params.SemesterID, name, slug, descriptionPtr, params.SortOrder, params.IsPublished).Scan(
		&created.ID,
		&created.SemesterID,
		&created.Name,
		&created.Slug,
		&created.Description,
		&created.SortOrder,
		&created.IsPublished,
		&created.CreatedAt,
		&created.UpdatedAt,
	)
	if err != nil {
		return Subject{}, fmt.Errorf("create subject: %w", err)
	}

	return created, nil
}

func (r *SubjectRepository) List(ctx context.Context) ([]Subject, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT
			sub.id,
			sub.semester_id,
			sem.name AS semester_name,
			c.name AS class_name,
			sub.name,
			sub.slug,
			sub.description,
			sub.sort_order,
			sub.is_published,
			sub.created_at,
			sub.updated_at
		FROM subjects sub
		JOIN semesters sem ON sem.id = sub.semester_id
		JOIN classes c ON c.id = sem.class_id
		ORDER BY
			c.sort_order ASC,
			c.name ASC,
			sem.sort_order ASC,
			sem.name ASC,
			sub.sort_order ASC,
			sub.name ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list subjects: %w", err)
	}
	defer rows.Close()

	subjects := make([]Subject, 0)

	for rows.Next() {
		var item Subject

		err := rows.Scan(
			&item.ID,
			&item.SemesterID,
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
			return nil, fmt.Errorf("scan subject: %w", err)
		}

		subjects = append(subjects, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate subjects: %w", err)
	}

	return subjects, nil
}

func (r *SubjectRepository) FindByID(ctx context.Context, id uuid.UUID) (Subject, error) {
	if id == uuid.Nil {
		return Subject{}, ErrSubjectNotFound
	}

	var item Subject

	err := r.pool.QueryRow(ctx, `
		SELECT
			sub.id,
			sub.semester_id,
			sem.name AS semester_name,
			c.name AS class_name,
			sub.name,
			sub.slug,
			sub.description,
			sub.sort_order,
			sub.is_published,
			sub.created_at,
			sub.updated_at
		FROM subjects sub
		JOIN semesters sem ON sem.id = sub.semester_id
		JOIN classes c ON c.id = sem.class_id
		WHERE sub.id = $1
	`, id).Scan(
		&item.ID,
		&item.SemesterID,
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
			return Subject{}, ErrSubjectNotFound
		}

		return Subject{}, fmt.Errorf("find subject by id: %w", err)
	}

	return item, nil
}

type UpdateSubjectParams struct {
	ID          uuid.UUID
	SemesterID  uuid.UUID
	Name        string
	Description string
	SortOrder   int
	IsPublished bool
}

func (r *SubjectRepository) Update(ctx context.Context, params UpdateSubjectParams) (Subject, error) {
	name := strings.TrimSpace(params.Name)
	description := strings.TrimSpace(params.Description)

	if params.ID == uuid.Nil {
		return Subject{}, ErrSubjectNotFound
	}

	if params.SemesterID == uuid.Nil {
		return Subject{}, errors.New("semester is required")
	}

	if name == "" {
		return Subject{}, errors.New("subject name is required")
	}

	slug := Slugify(name)
	if slug == "" {
		return Subject{}, errors.New("subject name must contain valid characters")
	}

	var descriptionPtr *string
	if description != "" {
		descriptionPtr = &description
	}

	var updated Subject

	err := r.pool.QueryRow(ctx, `
		UPDATE subjects
		SET
			semester_id = $2,
			name = $3,
			slug = $4,
			description = $5,
			sort_order = $6,
			is_published = $7
		WHERE id = $1
		RETURNING
			id,
			semester_id,
			name,
			slug,
			description,
			sort_order,
			is_published,
			created_at,
			updated_at
	`,
		params.ID,
		params.SemesterID,
		name,
		slug,
		descriptionPtr,
		params.SortOrder,
		params.IsPublished,
	).Scan(
		&updated.ID,
		&updated.SemesterID,
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
			return Subject{}, ErrSubjectNotFound
		}

		return Subject{}, fmt.Errorf("update subject: %w", err)
	}

	return updated, nil
}
