-- +goose Up
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE INDEX IF NOT EXISTS classes_slug_idx ON classes (slug);
CREATE INDEX IF NOT EXISTS semesters_class_id_slug_idx ON semesters (class_id, slug);
CREATE INDEX IF NOT EXISTS subjects_semester_id_slug_idx ON subjects (semester_id, slug);
CREATE INDEX IF NOT EXISTS units_subject_id_slug_idx ON units (subject_id, slug);
CREATE INDEX IF NOT EXISTS chapters_unit_id_slug_idx ON chapters (unit_id, slug);
CREATE INDEX IF NOT EXISTS notes_chapter_id_slug_idx ON notes (chapter_id, slug);

CREATE INDEX IF NOT EXISTS semesters_class_id_idx ON semesters (class_id);
CREATE INDEX IF NOT EXISTS subjects_semester_id_idx ON subjects (semester_id);
CREATE INDEX IF NOT EXISTS units_subject_id_idx ON units (subject_id);
CREATE INDEX IF NOT EXISTS chapters_unit_id_idx ON chapters (unit_id);
CREATE INDEX IF NOT EXISTS notes_chapter_id_idx ON notes (chapter_id);

CREATE INDEX IF NOT EXISTS classes_published_sort_idx
ON classes (is_published, sort_order, name);

CREATE INDEX IF NOT EXISTS semesters_published_sort_idx
ON semesters (class_id, is_published, sort_order, name);

CREATE INDEX IF NOT EXISTS subjects_published_sort_idx
ON subjects (semester_id, is_published, sort_order, name);

CREATE INDEX IF NOT EXISTS units_published_sort_idx
ON units (subject_id, is_published, sort_order, name);

CREATE INDEX IF NOT EXISTS chapters_published_sort_idx
ON chapters (unit_id, is_published, sort_order, name);

CREATE INDEX IF NOT EXISTS notes_public_listing_idx
ON notes (chapter_id, is_published, archived_at, sort_order, title);

CREATE INDEX IF NOT EXISTS notes_admin_listing_idx
ON notes (archived_at, created_at DESC, title);

CREATE INDEX IF NOT EXISTS notes_title_trgm_idx
ON notes USING gin (title gin_trgm_ops);

CREATE INDEX IF NOT EXISTS notes_description_trgm_idx
ON notes USING gin (description gin_trgm_ops);

CREATE INDEX IF NOT EXISTS notes_original_file_name_trgm_idx
ON notes USING gin (original_file_name gin_trgm_ops);

CREATE INDEX IF NOT EXISTS classes_name_trgm_idx
ON classes USING gin (name gin_trgm_ops);

CREATE INDEX IF NOT EXISTS semesters_name_trgm_idx
ON semesters USING gin (name gin_trgm_ops);

CREATE INDEX IF NOT EXISTS subjects_name_trgm_idx
ON subjects USING gin (name gin_trgm_ops);

CREATE INDEX IF NOT EXISTS units_name_trgm_idx
ON units USING gin (name gin_trgm_ops);

CREATE INDEX IF NOT EXISTS chapters_name_trgm_idx
ON chapters USING gin (name gin_trgm_ops);

-- +goose Down
DROP INDEX IF EXISTS chapters_name_trgm_idx;
DROP INDEX IF EXISTS units_name_trgm_idx;
DROP INDEX IF EXISTS subjects_name_trgm_idx;
DROP INDEX IF EXISTS semesters_name_trgm_idx;
DROP INDEX IF EXISTS classes_name_trgm_idx;

DROP INDEX IF EXISTS notes_original_file_name_trgm_idx;
DROP INDEX IF EXISTS notes_description_trgm_idx;
DROP INDEX IF EXISTS notes_title_trgm_idx;

DROP INDEX IF EXISTS notes_admin_listing_idx;
DROP INDEX IF EXISTS notes_public_listing_idx;

DROP INDEX IF EXISTS chapters_published_sort_idx;
DROP INDEX IF EXISTS units_published_sort_idx;
DROP INDEX IF EXISTS subjects_published_sort_idx;
DROP INDEX IF EXISTS semesters_published_sort_idx;
DROP INDEX IF EXISTS classes_published_sort_idx;

DROP INDEX IF EXISTS notes_chapter_id_idx;
DROP INDEX IF EXISTS chapters_unit_id_idx;
DROP INDEX IF EXISTS units_subject_id_idx;
DROP INDEX IF EXISTS subjects_semester_id_idx;
DROP INDEX IF EXISTS semesters_class_id_idx;

DROP INDEX IF EXISTS notes_chapter_id_slug_idx;
DROP INDEX IF EXISTS chapters_unit_id_slug_idx;
DROP INDEX IF EXISTS units_subject_id_slug_idx;
DROP INDEX IF EXISTS subjects_semester_id_slug_idx;
DROP INDEX IF EXISTS semesters_class_id_slug_idx;
DROP INDEX IF EXISTS classes_slug_idx;