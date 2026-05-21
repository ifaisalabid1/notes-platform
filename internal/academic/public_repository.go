package academic

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PublicRepository struct {
	pool *pgxpool.Pool
}

func NewPublicRepository(pool *pgxpool.Pool) *PublicRepository {
	return &PublicRepository{
		pool: pool,
	}
}

func (r *PublicRepository) PublishedClasses(ctx context.Context) ([]Class, error) {
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
		WHERE is_published = TRUE
		ORDER BY sort_order ASC, name ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list published classes: %w", err)
	}
	defer rows.Close()

	items := make([]Class, 0)

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

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate classes: %w", err)
	}

	return items, nil
}

func (r *PublicRepository) PublishedSemestersByClassSlug(ctx context.Context, classSlug string) (Class, []Semester, error) {
	var classItem Class

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
		WHERE slug = $1
		AND is_published = TRUE
	`, classSlug).Scan(
		&classItem.ID,
		&classItem.Name,
		&classItem.Slug,
		&classItem.Description,
		&classItem.SortOrder,
		&classItem.IsPublished,
		&classItem.CreatedAt,
		&classItem.UpdatedAt,
	)
	if err != nil {
		return Class{}, nil, fmt.Errorf("find published class: %w", err)
	}

	rows, err := r.pool.Query(ctx, `
		SELECT
			id,
			class_id,
			name,
			slug,
			description,
			sort_order,
			is_published,
			created_at,
			updated_at
		FROM semesters
		WHERE class_id = $1
		AND is_published = TRUE
		ORDER BY sort_order ASC, name ASC
	`, classItem.ID)
	if err != nil {
		return Class{}, nil, fmt.Errorf("list published semesters: %w", err)
	}
	defer rows.Close()

	semesters := make([]Semester, 0)

	for rows.Next() {
		var item Semester

		err := rows.Scan(
			&item.ID,
			&item.ClassID,
			&item.Name,
			&item.Slug,
			&item.Description,
			&item.SortOrder,
			&item.IsPublished,
			&item.CreatedAt,
			&item.UpdatedAt,
		)
		if err != nil {
			return Class{}, nil, fmt.Errorf("scan semester: %w", err)
		}

		item.ClassName = classItem.Name
		semesters = append(semesters, item)
	}

	if err := rows.Err(); err != nil {
		return Class{}, nil, fmt.Errorf("iterate semesters: %w", err)
	}

	return classItem, semesters, nil
}

func (r *PublicRepository) PublishedSubjects(ctx context.Context, classSlug string, semesterSlug string) (Class, Semester, []Subject, error) {
	classItem, semesterItem, err := r.findPublishedClassSemester(ctx, classSlug, semesterSlug)
	if err != nil {
		return Class{}, Semester{}, nil, err
	}

	rows, err := r.pool.Query(ctx, `
		SELECT
			id,
			semester_id,
			name,
			slug,
			description,
			sort_order,
			is_published,
			created_at,
			updated_at
		FROM subjects
		WHERE semester_id = $1
		AND is_published = TRUE
		ORDER BY sort_order ASC, name ASC
	`, semesterItem.ID)
	if err != nil {
		return Class{}, Semester{}, nil, fmt.Errorf("list published subjects: %w", err)
	}
	defer rows.Close()

	subjects := make([]Subject, 0)

	for rows.Next() {
		var item Subject

		err := rows.Scan(
			&item.ID,
			&item.SemesterID,
			&item.Name,
			&item.Slug,
			&item.Description,
			&item.SortOrder,
			&item.IsPublished,
			&item.CreatedAt,
			&item.UpdatedAt,
		)
		if err != nil {
			return Class{}, Semester{}, nil, fmt.Errorf("scan subject: %w", err)
		}

		item.ClassName = classItem.Name
		item.SemesterName = semesterItem.Name
		subjects = append(subjects, item)
	}

	if err := rows.Err(); err != nil {
		return Class{}, Semester{}, nil, fmt.Errorf("iterate subjects: %w", err)
	}

	return classItem, semesterItem, subjects, nil
}

func (r *PublicRepository) PublishedUnits(ctx context.Context, classSlug string, semesterSlug string, subjectSlug string) (Class, Semester, Subject, []Unit, error) {
	classItem, semesterItem, subjectItem, err := r.findPublishedClassSemesterSubject(ctx, classSlug, semesterSlug, subjectSlug)
	if err != nil {
		return Class{}, Semester{}, Subject{}, nil, err
	}

	rows, err := r.pool.Query(ctx, `
		SELECT
			id,
			subject_id,
			name,
			slug,
			description,
			sort_order,
			is_published,
			created_at,
			updated_at
		FROM units
		WHERE subject_id = $1
		AND is_published = TRUE
		ORDER BY sort_order ASC, name ASC
	`, subjectItem.ID)
	if err != nil {
		return Class{}, Semester{}, Subject{}, nil, fmt.Errorf("list published units: %w", err)
	}
	defer rows.Close()

	units := make([]Unit, 0)

	for rows.Next() {
		var item Unit

		err := rows.Scan(
			&item.ID,
			&item.SubjectID,
			&item.Name,
			&item.Slug,
			&item.Description,
			&item.SortOrder,
			&item.IsPublished,
			&item.CreatedAt,
			&item.UpdatedAt,
		)
		if err != nil {
			return Class{}, Semester{}, Subject{}, nil, fmt.Errorf("scan unit: %w", err)
		}

		item.ClassName = classItem.Name
		item.SemesterName = semesterItem.Name
		item.SubjectName = subjectItem.Name
		units = append(units, item)
	}

	if err := rows.Err(); err != nil {
		return Class{}, Semester{}, Subject{}, nil, fmt.Errorf("iterate units: %w", err)
	}

	return classItem, semesterItem, subjectItem, units, nil
}

func (r *PublicRepository) PublishedChapters(ctx context.Context, classSlug string, semesterSlug string, subjectSlug string, unitSlug string) (Class, Semester, Subject, Unit, []Chapter, error) {
	classItem, semesterItem, subjectItem, unitItem, err := r.findPublishedClassSemesterSubjectUnit(ctx, classSlug, semesterSlug, subjectSlug, unitSlug)
	if err != nil {
		return Class{}, Semester{}, Subject{}, Unit{}, nil, err
	}

	rows, err := r.pool.Query(ctx, `
		SELECT
			id,
			unit_id,
			name,
			slug,
			description,
			sort_order,
			is_published,
			created_at,
			updated_at
		FROM chapters
		WHERE unit_id = $1
		AND is_published = TRUE
		ORDER BY sort_order ASC, name ASC
	`, unitItem.ID)
	if err != nil {
		return Class{}, Semester{}, Subject{}, Unit{}, nil, fmt.Errorf("list published chapters: %w", err)
	}
	defer rows.Close()

	chapters := make([]Chapter, 0)

	for rows.Next() {
		var item Chapter

		err := rows.Scan(
			&item.ID,
			&item.UnitID,
			&item.Name,
			&item.Slug,
			&item.Description,
			&item.SortOrder,
			&item.IsPublished,
			&item.CreatedAt,
			&item.UpdatedAt,
		)
		if err != nil {
			return Class{}, Semester{}, Subject{}, Unit{}, nil, fmt.Errorf("scan chapter: %w", err)
		}

		item.ClassName = classItem.Name
		item.SemesterName = semesterItem.Name
		item.SubjectName = subjectItem.Name
		item.UnitName = unitItem.Name
		chapters = append(chapters, item)
	}

	if err := rows.Err(); err != nil {
		return Class{}, Semester{}, Subject{}, Unit{}, nil, fmt.Errorf("iterate chapters: %w", err)
	}

	return classItem, semesterItem, subjectItem, unitItem, chapters, nil
}

func (r *PublicRepository) PublishedNotes(ctx context.Context, classSlug string, semesterSlug string, subjectSlug string, unitSlug string, chapterSlug string) (Class, Semester, Subject, Unit, Chapter, []Note, error) {
	classItem, semesterItem, subjectItem, unitItem, chapterItem, err := r.findPublishedHierarchyToChapter(ctx, classSlug, semesterSlug, subjectSlug, unitSlug, chapterSlug)
	if err != nil {
		return Class{}, Semester{}, Subject{}, Unit{}, Chapter{}, nil, err
	}

	rows, err := r.pool.Query(ctx, `
		SELECT
			id,
			chapter_id,
			title,
			slug,
			description,
			original_file_name,
			stored_file_name,
			storage_key,
			content_type,
			file_size_bytes,
			is_pdf,
			is_watermarked,
			download_count,
			view_count,
			sort_order,
			is_published,
			uploaded_by,
			created_at,
			updated_at
		FROM notes
		WHERE chapter_id = $1
		AND is_published = TRUE
		ORDER BY sort_order ASC, title ASC
	`, chapterItem.ID)
	if err != nil {
		return Class{}, Semester{}, Subject{}, Unit{}, Chapter{}, nil, fmt.Errorf("list published notes: %w", err)
	}
	defer rows.Close()

	notes := make([]Note, 0)

	for rows.Next() {
		var item Note

		err := rows.Scan(
			&item.ID,
			&item.ChapterID,
			&item.Title,
			&item.Slug,
			&item.Description,
			&item.OriginalFileName,
			&item.StoredFileName,
			&item.StorageKey,
			&item.ContentType,
			&item.FileSizeBytes,
			&item.IsPDF,
			&item.IsWatermarked,
			&item.DownloadCount,
			&item.ViewCount,
			&item.SortOrder,
			&item.IsPublished,
			&item.UploadedBy,
			&item.CreatedAt,
			&item.UpdatedAt,
		)
		if err != nil {
			return Class{}, Semester{}, Subject{}, Unit{}, Chapter{}, nil, fmt.Errorf("scan note: %w", err)
		}

		item.ClassName = classItem.Name
		item.SemesterName = semesterItem.Name
		item.SubjectName = subjectItem.Name
		item.UnitName = unitItem.Name
		item.ChapterName = chapterItem.Name
		notes = append(notes, item)
	}

	if err := rows.Err(); err != nil {
		return Class{}, Semester{}, Subject{}, Unit{}, Chapter{}, nil, fmt.Errorf("iterate notes: %w", err)
	}

	return classItem, semesterItem, subjectItem, unitItem, chapterItem, notes, nil
}

func (r *PublicRepository) findPublishedClassSemester(ctx context.Context, classSlug string, semesterSlug string) (Class, Semester, error) {
	var classItem Class
	var semesterItem Semester

	err := r.pool.QueryRow(ctx, `
		SELECT
			c.id,
			c.name,
			c.slug,
			c.description,
			c.sort_order,
			c.is_published,
			c.created_at,
			c.updated_at,
			s.id,
			s.class_id,
			s.name,
			s.slug,
			s.description,
			s.sort_order,
			s.is_published,
			s.created_at,
			s.updated_at
		FROM semesters s
		JOIN classes c ON c.id = s.class_id
		WHERE c.slug = $1
		AND s.slug = $2
		AND c.is_published = TRUE
		AND s.is_published = TRUE
	`, classSlug, semesterSlug).Scan(
		&classItem.ID,
		&classItem.Name,
		&classItem.Slug,
		&classItem.Description,
		&classItem.SortOrder,
		&classItem.IsPublished,
		&classItem.CreatedAt,
		&classItem.UpdatedAt,
		&semesterItem.ID,
		&semesterItem.ClassID,
		&semesterItem.Name,
		&semesterItem.Slug,
		&semesterItem.Description,
		&semesterItem.SortOrder,
		&semesterItem.IsPublished,
		&semesterItem.CreatedAt,
		&semesterItem.UpdatedAt,
	)
	if err != nil {
		return Class{}, Semester{}, fmt.Errorf("find published class semester: %w", err)
	}

	semesterItem.ClassName = classItem.Name

	return classItem, semesterItem, nil
}

func (r *PublicRepository) findPublishedClassSemesterSubject(ctx context.Context, classSlug string, semesterSlug string, subjectSlug string) (Class, Semester, Subject, error) {
	classItem, semesterItem, err := r.findPublishedClassSemester(ctx, classSlug, semesterSlug)
	if err != nil {
		return Class{}, Semester{}, Subject{}, err
	}

	var subjectItem Subject

	err = r.pool.QueryRow(ctx, `
		SELECT
			id,
			semester_id,
			name,
			slug,
			description,
			sort_order,
			is_published,
			created_at,
			updated_at
		FROM subjects
		WHERE semester_id = $1
		AND slug = $2
		AND is_published = TRUE
	`, semesterItem.ID, subjectSlug).Scan(
		&subjectItem.ID,
		&subjectItem.SemesterID,
		&subjectItem.Name,
		&subjectItem.Slug,
		&subjectItem.Description,
		&subjectItem.SortOrder,
		&subjectItem.IsPublished,
		&subjectItem.CreatedAt,
		&subjectItem.UpdatedAt,
	)
	if err != nil {
		return Class{}, Semester{}, Subject{}, fmt.Errorf("find published subject: %w", err)
	}

	subjectItem.ClassName = classItem.Name
	subjectItem.SemesterName = semesterItem.Name

	return classItem, semesterItem, subjectItem, nil
}

func (r *PublicRepository) findPublishedClassSemesterSubjectUnit(ctx context.Context, classSlug string, semesterSlug string, subjectSlug string, unitSlug string) (Class, Semester, Subject, Unit, error) {
	classItem, semesterItem, subjectItem, err := r.findPublishedClassSemesterSubject(ctx, classSlug, semesterSlug, subjectSlug)
	if err != nil {
		return Class{}, Semester{}, Subject{}, Unit{}, err
	}

	var unitItem Unit

	err = r.pool.QueryRow(ctx, `
		SELECT
			id,
			subject_id,
			name,
			slug,
			description,
			sort_order,
			is_published,
			created_at,
			updated_at
		FROM units
		WHERE subject_id = $1
		AND slug = $2
		AND is_published = TRUE
	`, subjectItem.ID, unitSlug).Scan(
		&unitItem.ID,
		&unitItem.SubjectID,
		&unitItem.Name,
		&unitItem.Slug,
		&unitItem.Description,
		&unitItem.SortOrder,
		&unitItem.IsPublished,
		&unitItem.CreatedAt,
		&unitItem.UpdatedAt,
	)
	if err != nil {
		return Class{}, Semester{}, Subject{}, Unit{}, fmt.Errorf("find published unit: %w", err)
	}

	unitItem.ClassName = classItem.Name
	unitItem.SemesterName = semesterItem.Name
	unitItem.SubjectName = subjectItem.Name

	return classItem, semesterItem, subjectItem, unitItem, nil
}

func (r *PublicRepository) findPublishedHierarchyToChapter(ctx context.Context, classSlug string, semesterSlug string, subjectSlug string, unitSlug string, chapterSlug string) (Class, Semester, Subject, Unit, Chapter, error) {
	classItem, semesterItem, subjectItem, unitItem, err := r.findPublishedClassSemesterSubjectUnit(ctx, classSlug, semesterSlug, subjectSlug, unitSlug)
	if err != nil {
		return Class{}, Semester{}, Subject{}, Unit{}, Chapter{}, err
	}

	var chapterItem Chapter

	err = r.pool.QueryRow(ctx, `
		SELECT
			id,
			unit_id,
			name,
			slug,
			description,
			sort_order,
			is_published,
			created_at,
			updated_at
		FROM chapters
		WHERE unit_id = $1
		AND slug = $2
		AND is_published = TRUE
	`, unitItem.ID, chapterSlug).Scan(
		&chapterItem.ID,
		&chapterItem.UnitID,
		&chapterItem.Name,
		&chapterItem.Slug,
		&chapterItem.Description,
		&chapterItem.SortOrder,
		&chapterItem.IsPublished,
		&chapterItem.CreatedAt,
		&chapterItem.UpdatedAt,
	)
	if err != nil {
		return Class{}, Semester{}, Subject{}, Unit{}, Chapter{}, fmt.Errorf("find published chapter: %w", err)
	}

	chapterItem.ClassName = classItem.Name
	chapterItem.SemesterName = semesterItem.Name
	chapterItem.SubjectName = subjectItem.Name
	chapterItem.UnitName = unitItem.Name

	return classItem, semesterItem, subjectItem, unitItem, chapterItem, nil
}

func (r *PublicRepository) IncrementNoteViewCount(ctx context.Context, noteID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE notes
		SET view_count = view_count + 1
		WHERE id = $1
	`, noteID)
	if err != nil {
		return fmt.Errorf("increment note view count: %w", err)
	}

	return nil
}
