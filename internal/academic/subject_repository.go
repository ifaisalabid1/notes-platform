package academic

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

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
