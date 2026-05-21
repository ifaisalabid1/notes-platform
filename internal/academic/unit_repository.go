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

type Unit struct {
	ID           uuid.UUID
	SubjectID    uuid.UUID
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

type UnitRepository struct {
	pool *pgxpool.Pool
}

func NewUnitRepository(pool *pgxpool.Pool) *UnitRepository {
	return &UnitRepository{
		pool: pool,
	}
}

type CreateUnitParams struct {
	SubjectID   uuid.UUID
	Name        string
	Description string
	SortOrder   int
	IsPublished bool
}

func (r *UnitRepository) Create(ctx context.Context, params CreateUnitParams) (Unit, error) {
	name := strings.TrimSpace(params.Name)
	description := strings.TrimSpace(params.Description)

	if params.SubjectID == uuid.Nil {
		return Unit{}, errors.New("subject is required")
	}

	if name == "" {
		return Unit{}, errors.New("unit name is required")
	}

	slug := Slugify(name)
	if slug == "" {
		return Unit{}, errors.New("unit name must contain valid characters")
	}

	var descriptionPtr *string
	if description != "" {
		descriptionPtr = &description
	}

	var created Unit

	err := r.pool.QueryRow(ctx, `
		INSERT INTO units (
			subject_id,
			name,
			slug,
			description,
			sort_order,
			is_published
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING
			id,
			subject_id,
			name,
			slug,
			description,
			sort_order,
			is_published,
			created_at,
			updated_at
	`, params.SubjectID, name, slug, descriptionPtr, params.SortOrder, params.IsPublished).Scan(
		&created.ID,
		&created.SubjectID,
		&created.Name,
		&created.Slug,
		&created.Description,
		&created.SortOrder,
		&created.IsPublished,
		&created.CreatedAt,
		&created.UpdatedAt,
	)
	if err != nil {
		return Unit{}, fmt.Errorf("create unit: %w", err)
	}

	return created, nil
}

func (r *UnitRepository) List(ctx context.Context) ([]Unit, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT
			u.id,
			u.subject_id,
			sub.name AS subject_name,
			sem.name AS semester_name,
			c.name AS class_name,
			u.name,
			u.slug,
			u.description,
			u.sort_order,
			u.is_published,
			u.created_at,
			u.updated_at
		FROM units u
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
			u.name ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list units: %w", err)
	}
	defer rows.Close()

	units := make([]Unit, 0)

	for rows.Next() {
		var item Unit

		err := rows.Scan(
			&item.ID,
			&item.SubjectID,
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
			return nil, fmt.Errorf("scan unit: %w", err)
		}

		units = append(units, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate units: %w", err)
	}

	return units, nil
}
