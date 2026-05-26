package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Log struct {
	ID         uuid.UUID
	AdminID    *uuid.UUID
	Action     string
	EntityType string
	EntityID   *uuid.UUID
	Message    string
	Metadata   map[string]any
	IPAddress  *string
	UserAgent  *string
	CreatedAt  time.Time
}

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{
		pool: pool,
	}
}

type CreateLogParams struct {
	AdminID    *uuid.UUID
	Action     string
	EntityType string
	EntityID   *uuid.UUID
	Message    string
	Metadata   map[string]any
	IPAddress  string
	UserAgent  string
}

func (r *Repository) Create(ctx context.Context, params CreateLogParams) error {
	action := strings.TrimSpace(params.Action)
	entityType := strings.TrimSpace(params.EntityType)
	message := strings.TrimSpace(params.Message)

	if action == "" {
		return fmt.Errorf("audit action is required")
	}

	if entityType == "" {
		return fmt.Errorf("audit entity type is required")
	}

	if message == "" {
		return fmt.Errorf("audit message is required")
	}

	metadata := params.Metadata
	if metadata == nil {
		metadata = map[string]any{}
	}

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("marshal audit metadata: %w", err)
	}

	var ipAddress *string
	if strings.TrimSpace(params.IPAddress) != "" {
		value := strings.TrimSpace(params.IPAddress)
		ipAddress = &value
	}

	var userAgent *string
	if strings.TrimSpace(params.UserAgent) != "" {
		value := strings.TrimSpace(params.UserAgent)
		userAgent = &value
	}

	_, err = r.pool.Exec(ctx, `
		INSERT INTO admin_audit_logs (
			admin_id,
			action,
			entity_type,
			entity_id,
			message,
			metadata,
			ip_address,
			user_agent
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`,
		params.AdminID,
		action,
		entityType,
		params.EntityID,
		message,
		metadataJSON,
		ipAddress,
		userAgent,
	)
	if err != nil {
		return fmt.Errorf("create audit log: %w", err)
	}

	return nil
}

type ListParams struct {
	Limit  int
	Offset int
}

type PaginatedLogs struct {
	Logs       []Log
	TotalCount int
}

func (r *Repository) List(ctx context.Context, params ListParams) (PaginatedLogs, error) {
	limit := params.Limit
	if limit <= 0 {
		limit = 50
	}

	if limit > 100 {
		limit = 100
	}

	offset := params.Offset
	if offset < 0 {
		offset = 0
	}

	var totalCount int

	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM admin_audit_logs
	`).Scan(&totalCount)
	if err != nil {
		return PaginatedLogs{}, fmt.Errorf("count audit logs: %w", err)
	}

	rows, err := r.pool.Query(ctx, `
		SELECT
			id,
			admin_id,
			action,
			entity_type,
			entity_id,
			message,
			metadata,
			ip_address,
			user_agent,
			created_at
		FROM admin_audit_logs
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return PaginatedLogs{}, fmt.Errorf("list audit logs: %w", err)
	}
	defer rows.Close()

	logs := make([]Log, 0)

	for rows.Next() {
		var item Log
		var metadataJSON []byte

		err := rows.Scan(
			&item.ID,
			&item.AdminID,
			&item.Action,
			&item.EntityType,
			&item.EntityID,
			&item.Message,
			&metadataJSON,
			&item.IPAddress,
			&item.UserAgent,
			&item.CreatedAt,
		)
		if err != nil {
			return PaginatedLogs{}, fmt.Errorf("scan audit log: %w", err)
		}

		item.Metadata = map[string]any{}
		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &item.Metadata); err != nil {
				return PaginatedLogs{}, fmt.Errorf("unmarshal audit metadata: %w", err)
			}
		}

		logs = append(logs, item)
	}

	if err := rows.Err(); err != nil {
		return PaginatedLogs{}, fmt.Errorf("iterate audit logs: %w", err)
	}

	return PaginatedLogs{
		Logs:       logs,
		TotalCount: totalCount,
	}, nil
}
