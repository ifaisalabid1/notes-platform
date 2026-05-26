package dashboard

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Stats struct {
	Classes       int
	Semesters     int
	Subjects      int
	Units         int
	Chapters      int
	ActiveNotes   int
	ArchivedNotes int
	Admins        int
	AuditEvents   int
}

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{
		pool: pool,
	}
}

func (r *Repository) Stats(ctx context.Context) (Stats, error) {
	var stats Stats

	err := r.pool.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*) FROM classes),
			(SELECT COUNT(*) FROM semesters),
			(SELECT COUNT(*) FROM subjects),
			(SELECT COUNT(*) FROM units),
			(SELECT COUNT(*) FROM chapters),
			(SELECT COUNT(*) FROM notes WHERE archived_at IS NULL),
			(SELECT COUNT(*) FROM notes WHERE archived_at IS NOT NULL),
			(SELECT COUNT(*) FROM admins),
			(SELECT COUNT(*) FROM admin_audit_logs)
	`).Scan(
		&stats.Classes,
		&stats.Semesters,
		&stats.Subjects,
		&stats.Units,
		&stats.Chapters,
		&stats.ActiveNotes,
		&stats.ArchivedNotes,
		&stats.Admins,
		&stats.AuditEvents,
	)
	if err != nil {
		return Stats{}, fmt.Errorf("load dashboard stats: %w", err)
	}

	return stats, nil
}
