-- +goose Up
ALTER TABLE marking_jobs ADD COLUMN completed_at TIMESTAMP;

UPDATE marking_jobs
SET completed_at = CURRENT_TIMESTAMP
WHERE status IN ('success', 'failed');

-- +goose Down
ALTER TABLE marking_jobs DROP COLUMN completed_at;
