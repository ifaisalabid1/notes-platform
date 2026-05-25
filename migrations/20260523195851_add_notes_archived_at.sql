-- +goose Up
ALTER TABLE notes
ADD COLUMN IF NOT EXISTS archived_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS notes_archived_at_idx ON notes (archived_at);

-- +goose Down
DROP INDEX IF EXISTS notes_archived_at_idx;

ALTER TABLE notes
DROP COLUMN IF EXISTS archived_at;