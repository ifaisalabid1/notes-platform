package dashboard

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
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

type PopularNote struct {
	ID               uuid.UUID
	Title            string
	OriginalFileName string
	ClassName        string
	SemesterName     string
	SubjectName      string
	UnitName         string
	ChapterName      string
	ViewCount        int
	IsPublished      bool
	ArchivedAt       *time.Time
	CreatedAt        time.Time
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

func (r *Repository) PopularNotes(ctx context.Context, limit int) ([]PopularNote, error) {
	if limit <= 0 {
		limit = 5
	}

	if limit > 20 {
		limit = 20
	}

	rows, err := r.pool.Query(ctx, `
		SELECT
			n.id,
			n.title,
			n.original_file_name,
			c.name AS class_name,
			sem.name AS semester_name,
			sub.name AS subject_name,
			u.name AS unit_name,
			ch.name AS chapter_name,
			n.view_count,
			n.is_published,
			n.archived_at,
			n.created_at
		FROM notes n
		JOIN chapters ch ON ch.id = n.chapter_id
		JOIN units u ON u.id = ch.unit_id
		JOIN subjects sub ON sub.id = u.subject_id
		JOIN semesters sem ON sem.id = sub.semester_id
		JOIN classes c ON c.id = sem.class_id
		ORDER BY
			n.view_count DESC,
			n.created_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("list popular notes: %w", err)
	}
	defer rows.Close()

	notes := make([]PopularNote, 0)

	for rows.Next() {
		var note PopularNote

		err := rows.Scan(
			&note.ID,
			&note.Title,
			&note.OriginalFileName,
			&note.ClassName,
			&note.SemesterName,
			&note.SubjectName,
			&note.UnitName,
			&note.ChapterName,
			&note.ViewCount,
			&note.IsPublished,
			&note.ArchivedAt,
			&note.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan popular note: %w", err)
		}

		notes = append(notes, note)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate popular notes: %w", err)
	}

	return notes, nil
}
