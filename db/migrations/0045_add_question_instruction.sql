-- +goose Up
ALTER TABLE questions ADD COLUMN instruction TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE questions DROP COLUMN instruction;
