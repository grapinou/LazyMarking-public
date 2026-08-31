-- +goose Up
ALTER TABLE marking_jobs
ADD COLUMN exam_generated_id INTEGER
REFERENCES exams_generated(id) ON DELETE RESTRICT;

-- Existing rows remain NULL because their generation cannot be inferred reliably.
-- Every new row must instead be linked to a generation owned by the same user.
CREATE TRIGGER marking_jobs_generation_owner_insert
BEFORE INSERT ON marking_jobs
WHEN NEW.exam_generated_id IS NULL
  OR NOT EXISTS (
      SELECT 1
      FROM exams_generated
      WHERE id = NEW.exam_generated_id
        AND user_id = NEW.user_id
  )
BEGIN
    SELECT RAISE(ABORT, 'generated exam must belong to marking job user');
END;

CREATE TRIGGER marking_jobs_generation_owner_update
BEFORE UPDATE OF exam_generated_id, user_id ON marking_jobs
WHEN NEW.exam_generated_id IS NULL
  OR NOT EXISTS (
      SELECT 1
      FROM exams_generated
      WHERE id = NEW.exam_generated_id
        AND user_id = NEW.user_id
  )
BEGIN
    SELECT RAISE(ABORT, 'generated exam must belong to marking job user');
END;

-- +goose Down
DROP TRIGGER marking_jobs_generation_owner_update;
DROP TRIGGER marking_jobs_generation_owner_insert;
ALTER TABLE marking_jobs DROP COLUMN exam_generated_id;
