package admin

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

var (
	ErrOwnerAlreadyExists = errors.New("owner super admin already exists")
	ErrAdminNotFound      = errors.New("admin not found")
)

type Role string

const (
	RoleSuperAdmin Role = "super_admin"
	RoleAdmin      Role = "admin"
)

type Admin struct {
	ID           uuid.UUID
	Name         string
	Email        string
	PasswordHash string
	Role         Role
	IsOwner      bool
	CreatedBy    *uuid.UUID
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{
		pool: pool,
	}
}

type CreateOwnerParams struct {
	Name         string
	Email        string
	PasswordHash string
}

type CreateAdminParams struct {
	Name         string
	Email        string
	PasswordHash string
	CreatedBy    uuid.UUID
}

func (r *Repository) CreateOwner(ctx context.Context, params CreateOwnerParams) (Admin, error) {
	name := strings.TrimSpace(params.Name)
	email := normalizeEmail(params.Email)

	if name == "" {
		return Admin{}, errors.New("name is required")
	}

	if email == "" {
		return Admin{}, errors.New("email is required")
	}

	if params.PasswordHash == "" {
		return Admin{}, errors.New("password hash is required")
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Admin{}, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var ownerExists bool
	err = tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM admins
			WHERE is_owner = TRUE
			)
	`).Scan(&ownerExists)
	if err != nil {
		return Admin{}, fmt.Errorf("check owner exists: %w", err)
	}

	if ownerExists {
		return Admin{}, ErrOwnerAlreadyExists
	}

	var created Admin

	err = tx.QueryRow(ctx, `
		INSERT INTO admins (
			name,
			email,
			password_hash,
			role,
			is_owner
		)
		VALUES ($1, $2, $3, $4, TRUE)
		RETURNING
			id,
			name,
			email,
			password_hash,
			role,
			is_owner,
			created_by,
			created_at,
			updated_at
	`, name, email, params.PasswordHash, RoleSuperAdmin).Scan(
		&created.ID,
		&created.Name,
		&created.Email,
		&created.PasswordHash,
		&created.Role,
		&created.IsOwner,
		&created.CreatedBy,
		&created.CreatedAt,
		&created.UpdatedAt,
	)
	if err != nil {
		return Admin{}, fmt.Errorf("create owner: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return Admin{}, fmt.Errorf("commit transaction: %w", err)
	}

	return created, nil
}

func (r *Repository) CountOwners(ctx context.Context) (int, error) {
	var count int

	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM admins
		WHERE is_owner = TRUE
	`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count owners: %w", err)
	}

	return count, nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func (r *Repository) FindByEmail(ctx context.Context, email string) (Admin, error) {
	normalizedEmail := normalizeEmail(email)

	if normalizedEmail == "" {
		return Admin{}, ErrAdminNotFound
	}

	var found Admin

	err := r.pool.QueryRow(ctx, `
		SELECT
			id,
			name,
			email,
			password_hash,
			role,
			is_owner,
			created_by,
			created_at,
			updated_at
		FROM admins
		WHERE email = $1
	`, normalizedEmail).Scan(
		&found.ID,
		&found.Name,
		&found.Email,
		&found.PasswordHash,
		&found.Role,
		&found.IsOwner,
		&found.CreatedBy,
		&found.CreatedAt,
		&found.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Admin{}, ErrAdminNotFound
		}

		return Admin{}, fmt.Errorf("find admin by email: %w", err)
	}

	return found, nil
}

func (r *Repository) CreateAdmin(ctx context.Context, params CreateAdminParams) (Admin, error) {
	name := strings.TrimSpace(params.Name)
	email := normalizeEmail(params.Email)

	if name == "" {
		return Admin{}, errors.New("name is required")
	}

	if email == "" {
		return Admin{}, errors.New("email is required")
	}

	if params.PasswordHash == "" {
		return Admin{}, errors.New("password hash is required")
	}

	if params.CreatedBy == uuid.Nil {
		return Admin{}, errors.New("created by is required")
	}

	var created Admin

	err := r.pool.QueryRow(ctx, `
		INSERT INTO admins (
			name,
			email,
			password_hash,
			role,
			is_owner,
			created_by
		)
		VALUES ($1, $2, $3, $4, FALSE, $5)
		RETURNING
			id,
			name,
			email,
			password_hash,
			role,
			is_owner,
			created_by,
			created_at,
			updated_at
	`, name, email, params.PasswordHash, RoleAdmin, params.CreatedBy).Scan(
		&created.ID,
		&created.Name,
		&created.Email,
		&created.PasswordHash,
		&created.Role,
		&created.IsOwner,
		&created.CreatedBy,
		&created.CreatedAt,
		&created.UpdatedAt,
	)
	if err != nil {
		return Admin{}, fmt.Errorf("create admin: %w", err)
	}

	return created, nil
}

func (r *Repository) List(ctx context.Context) ([]Admin, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT
			id,
			name,
			email,
			password_hash,
			role,
			is_owner,
			created_by,
			created_at,
			updated_at
		FROM admins
		ORDER BY is_owner DESC, created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("list admins: %w", err)
	}
	defer rows.Close()

	admins := make([]Admin, 0)

	for rows.Next() {
		var item Admin

		err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.Email,
			&item.PasswordHash,
			&item.Role,
			&item.IsOwner,
			&item.CreatedBy,
			&item.CreatedAt,
			&item.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan admin: %w", err)
		}

		admins = append(admins, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate admins: %w", err)
	}

	return admins, nil
}
