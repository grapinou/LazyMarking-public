-- +goose Up
ALTER TABLE marking_jobs ADD COLUMN result_schema_version INTEGER;
ALTER TABLE marking_jobs ADD COLUMN marking_algorithm_version TEXT;
ALTER TABLE marking_jobs ADD COLUMN detection_threshold REAL;

CREATE TRIGGER marking_jobs_result_metadata_insert
BEFORE INSERT ON marking_jobs
WHEN NOT (
    (NEW.result_schema_version IS NULL
        AND NEW.marking_algorithm_version IS NULL
        AND NEW.detection_threshold IS NULL)
    OR
    (NEW.result_schema_version IS NOT NULL
        AND NEW.result_schema_version >= 1
        AND NEW.marking_algorithm_version IS NOT NULL
        AND length(NEW.marking_algorithm_version) > 0
        AND NEW.detection_threshold IS NOT NULL
        AND NEW.detection_threshold >= 0
        AND NEW.detection_threshold <= 255)
)
BEGIN
    SELECT RAISE(ABORT, 'marking result metadata must be complete and valid');
END;

CREATE TRIGGER marking_jobs_result_metadata_update
BEFORE UPDATE OF result_schema_version, marking_algorithm_version, detection_threshold ON marking_jobs
WHEN NOT (
    (NEW.result_schema_version IS NULL
        AND NEW.marking_algorithm_version IS NULL
        AND NEW.detection_threshold IS NULL)
    OR
    (NEW.result_schema_version IS NOT NULL
        AND NEW.result_schema_version >= 1
        AND NEW.marking_algorithm_version IS NOT NULL
        AND length(NEW.marking_algorithm_version) > 0
        AND NEW.detection_threshold IS NOT NULL
        AND NEW.detection_threshold >= 0
        AND NEW.detection_threshold <= 255)
)
BEGIN
    SELECT RAISE(ABORT, 'marking result metadata must be complete and valid');
END;

CREATE TRIGGER marking_jobs_result_identity_immutable
BEFORE UPDATE OF exam_generated_id, result_schema_version, marking_algorithm_version, detection_threshold ON marking_jobs
WHEN OLD.exam_generated_id IS NOT NEW.exam_generated_id
  OR OLD.result_schema_version IS NOT NEW.result_schema_version
  OR OLD.marking_algorithm_version IS NOT NEW.marking_algorithm_version
  OR OLD.detection_threshold IS NOT NEW.detection_threshold
BEGIN
    SELECT RAISE(ABORT, 'marking job result identity is immutable');
END;

-- +goose Down
DROP TRIGGER marking_jobs_result_identity_immutable;
DROP TRIGGER marking_jobs_result_metadata_update;
DROP TRIGGER marking_jobs_result_metadata_insert;
ALTER TABLE marking_jobs DROP COLUMN detection_threshold;
ALTER TABLE marking_jobs DROP COLUMN marking_algorithm_version;
ALTER TABLE marking_jobs DROP COLUMN result_schema_version;
