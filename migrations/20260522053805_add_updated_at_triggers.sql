-- +goose Up

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER admins_set_updated_at
BEFORE UPDATE ON admins
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER classes_set_updated_at
BEFORE UPDATE ON classes
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER semesters_set_updated_at
BEFORE UPDATE ON semesters
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER subjects_set_updated_at
BEFORE UPDATE ON subjects
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER units_set_updated_at
BEFORE UPDATE ON units
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER chapters_set_updated_at
BEFORE UPDATE ON chapters
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER notes_set_updated_at
BEFORE UPDATE ON notes
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

-- +goose Down
DROP TRIGGER IF EXISTS notes_set_updated_at ON notes;
DROP TRIGGER IF EXISTS chapters_set_updated_at ON chapters;
DROP TRIGGER IF EXISTS units_set_updated_at ON units;
DROP TRIGGER IF EXISTS subjects_set_updated_at ON subjects;
DROP TRIGGER IF EXISTS semesters_set_updated_at ON semesters;
DROP TRIGGER IF EXISTS classes_set_updated_at ON classes;
DROP TRIGGER IF EXISTS admins_set_updated_at ON admins;

DROP FUNCTION IF EXISTS set_updated_at();