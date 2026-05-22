package academic

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrClassNotFound = errors.New("class not found")

type Class struct {
	ID          uuid.UUID
	Name        string
	Slug        string
	Description *string
	SortOrder   int
	IsPublished bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type ClassRepository struct {
	pool *pgxpool.Pool
}

func NewClassRepository(pool *pgxpool.Pool) *ClassRepository {
	return &ClassRepository{
		pool: pool,
	}
}

type CreateClassParams struct {
	Name        string
	Description string
	SortOrder   int
	IsPublished bool
}

func (r *ClassRepository) Create(ctx context.Context, params CreateClassParams) (Class, error) {
	name := strings.TrimSpace(params.Name)
	description := strings.TrimSpace(params.Description)

	if name == "" {
		return Class{}, errors.New("class name is required")
	}

	slug := Slugify(name)

	if slug == "" {
		return Class{}, errors.New("class name must contain valid characters")
	}

	var descriptionPtr *string
	if description != "" {
		descriptionPtr = &description
	}

	var created Class

	err := r.pool.QueryRow(ctx, `
		INSERT INTO classes (
			name,
			slug,
			description,
			sort_order,
			is_published
		)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING
			id,
			name,
			slug,
			description,
			sort_order,
			is_published,
			created_at,
			updated_at
	`, name, slug, descriptionPtr, params.SortOrder, params.IsPublished).Scan(
		&created.ID,
		&created.Name,
		&created.Slug,
		&created.Description,
		&created.SortOrder,
		&created.IsPublished,
		&created.CreatedAt,
		&created.UpdatedAt,
	)
	if err != nil {
		return Class{}, fmt.Errorf("create class: %w", err)
	}

	return created, nil
}

func (r *ClassRepository) List(ctx context.Context) ([]Class, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT
			id,
			name,
			slug,
			description,
			sort_order,
			is_published,
			created_at,
			updated_at
		FROM classes
		ORDER BY sort_order ASC, name ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list classes: %w", err)
	}
	defer rows.Close()

	classes := make([]Class, 0)

	for rows.Next() {
		var item Class

		err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.Slug,
			&item.Description,
			&item.SortOrder,
			&item.IsPublished,
			&item.CreatedAt,
			&item.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan class: %w", err)
		}

		classes = append(classes, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate classes: %w", err)
	}

	return classes, nil
}

func Slugify(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))

	var builder strings.Builder
	previousDash := false

	for _, r := range value {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			builder.WriteRune(r)
			previousDash = false
		case unicode.IsSpace(r) || r == '-' || r == '_' || r == '.':
			if !previousDash {
				builder.WriteRune('-')
				previousDash = true
			}
		}
	}

	slug := builder.String()
	slug = strings.Trim(slug, "-")

	re := regexp.MustCompile(`-+`)
	slug = re.ReplaceAllString(slug, "-")

	return slug
}

func (r *ClassRepository) FindByID(ctx context.Context, id uuid.UUID) (Class, error) {
	if id == uuid.Nil {
		return Class{}, ErrClassNotFound
	}

	var item Class

	err := r.pool.QueryRow(ctx, `
		SELECT
			id,
			name,
			slug,
			description,
			sort_order,
			is_published,
			created_at,
			updated_at
		FROM classes
		WHERE id = $1
	`, id).Scan(
		&item.ID,
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
			return Class{}, ErrClassNotFound
		}

		return Class{}, fmt.Errorf("find class by id: %w", err)
	}

	return item, nil
}

type UpdateClassParams struct {
	ID          uuid.UUID
	Name        string
	Description string
	SortOrder   int
	IsPublished bool
}

func (r *ClassRepository) Update(ctx context.Context, params UpdateClassParams) (Class, error) {
	name := strings.TrimSpace(params.Name)
	description := strings.TrimSpace(params.Description)

	if params.ID == uuid.Nil {
		return Class{}, ErrClassNotFound
	}

	if name == "" {
		return Class{}, errors.New("class name is required")
	}

	slug := Slugify(name)
	if slug == "" {
		return Class{}, errors.New("class name must contain valid characters")
	}

	var descriptionPtr *string
	if description != "" {
		descriptionPtr = &description
	}

	var updated Class

	err := r.pool.QueryRow(ctx, `
		UPDATE classes
		SET
			name = $2,
			slug = $3,
			description = $4,
			sort_order = $5,
			is_published = $6
		WHERE id = $1
		RETURNING
			id,
			name,
			slug,
			description,
			sort_order,
			is_published,
			created_at,
			updated_at
	`,
		params.ID,
		name,
		slug,
		descriptionPtr,
		params.SortOrder,
		params.IsPublished,
	).Scan(
		&updated.ID,
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
			return Class{}, ErrClassNotFound
		}

		return Class{}, fmt.Errorf("update class: %w", err)
	}

	return updated, nil
}
