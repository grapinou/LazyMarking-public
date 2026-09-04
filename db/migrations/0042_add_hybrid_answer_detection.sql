-- +goose Up
ALTER TABLE marking_jobs ADD COLUMN review_policy_version TEXT;
ALTER TABLE marking_jobs ADD COLUMN v2_roi_radius_ratio REAL;
ALTER TABLE marking_jobs ADD COLUMN v2_dark_pixel_threshold REAL;
ALTER TABLE marking_jobs ADD COLUMN v2_dark_ratio_threshold REAL;
ALTER TABLE marking_jobs ADD COLUMN v2_chroma_pixel_threshold REAL;
ALTER TABLE marking_jobs ADD COLUMN v2_chroma_ratio_threshold REAL;

ALTER TABLE marking_answer_detections ADD COLUMN historical_state INTEGER CHECK (historical_state IN (0, 1));
ALTER TABLE marking_answer_detections ADD COLUMN v2_state INTEGER CHECK (v2_state IN (0, 1));
ALTER TABLE marking_answer_detections ADD COLUMN dark_ratio REAL CHECK (dark_ratio >= 0 AND dark_ratio <= 1);
ALTER TABLE marking_answer_detections ADD COLUMN chroma_ratio REAL CHECK (chroma_ratio >= 0 AND chroma_ratio <= 1);
ALTER TABLE marking_answer_detections ADD COLUMN grayscale_signal INTEGER CHECK (grayscale_signal IN (0, 1));
ALTER TABLE marking_answer_detections ADD COLUMN color_signal INTEGER CHECK (color_signal IN (0, 1));
ALTER TABLE marking_answer_detections ADD COLUMN automatic_state INTEGER CHECK (automatic_state IN (0, 1));
ALTER TABLE marking_answer_detections ADD COLUMN review_reason TEXT CHECK (review_reason IS NULL OR review_reason = 'detector_disagreement');

-- +goose StatementBegin
CREATE TRIGGER marking_answer_detections_hybrid_insert
BEFORE INSERT ON marking_answer_detections
WHEN NOT (
    (NEW.historical_state IS NULL AND NEW.v2_state IS NULL
      AND NEW.dark_ratio IS NULL AND NEW.chroma_ratio IS NULL
      AND NEW.grayscale_signal IS NULL AND NEW.color_signal IS NULL
      AND NEW.automatic_state IS NULL
      AND NEW.review_reason IS NULL)
    OR
    (NEW.historical_state IS NOT NULL AND NEW.v2_state IS NOT NULL
      AND NEW.dark_ratio IS NOT NULL AND NEW.chroma_ratio IS NOT NULL
      AND NEW.grayscale_signal IS NOT NULL AND NEW.color_signal IS NOT NULL
      AND NEW.detected_state = NEW.historical_state
      AND NEW.v2_state = (NEW.grayscale_signal OR NEW.color_signal)
      AND ((NEW.historical_state = NEW.v2_state AND NEW.automatic_state = NEW.historical_state AND NEW.review_reason IS NULL)
        OR (NEW.historical_state != NEW.v2_state AND NEW.automatic_state IS NULL AND NEW.review_reason = 'detector_disagreement')))
)
BEGIN
    SELECT RAISE(ABORT, 'hybrid answer detection metadata must be complete and coherent');
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER marking_jobs_hybrid_parameters_insert
BEFORE INSERT ON marking_jobs
WHEN NOT (
    (NEW.review_policy_version IS NULL AND NEW.v2_roi_radius_ratio IS NULL
      AND NEW.v2_dark_pixel_threshold IS NULL AND NEW.v2_dark_ratio_threshold IS NULL
      AND NEW.v2_chroma_pixel_threshold IS NULL AND NEW.v2_chroma_ratio_threshold IS NULL)
    OR
    (NEW.review_policy_version = 'detector-agreement-v1'
      AND NEW.v2_roi_radius_ratio > 0 AND NEW.v2_roi_radius_ratio <= 1
      AND NEW.v2_dark_pixel_threshold >= 0 AND NEW.v2_dark_pixel_threshold <= 255
      AND NEW.v2_dark_ratio_threshold >= 0 AND NEW.v2_dark_ratio_threshold <= 1
      AND NEW.v2_chroma_pixel_threshold >= 0 AND NEW.v2_chroma_pixel_threshold <= 255
      AND NEW.v2_chroma_ratio_threshold >= 0 AND NEW.v2_chroma_ratio_threshold <= 1)
)
BEGIN
    SELECT RAISE(ABORT, 'hybrid marking parameters must be complete and valid');
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER marking_jobs_hybrid_parameters_immutable
BEFORE UPDATE OF review_policy_version, v2_roi_radius_ratio, v2_dark_pixel_threshold,
  v2_dark_ratio_threshold, v2_chroma_pixel_threshold, v2_chroma_ratio_threshold ON marking_jobs
WHEN OLD.review_policy_version IS NOT NEW.review_policy_version
  OR OLD.v2_roi_radius_ratio IS NOT NEW.v2_roi_radius_ratio
  OR OLD.v2_dark_pixel_threshold IS NOT NEW.v2_dark_pixel_threshold
  OR OLD.v2_dark_ratio_threshold IS NOT NEW.v2_dark_ratio_threshold
  OR OLD.v2_chroma_pixel_threshold IS NOT NEW.v2_chroma_pixel_threshold
  OR OLD.v2_chroma_ratio_threshold IS NOT NEW.v2_chroma_ratio_threshold
BEGIN
    SELECT RAISE(ABORT, 'hybrid marking parameters are immutable');
END;
-- +goose StatementEnd

-- +goose Down
DROP TRIGGER marking_jobs_hybrid_parameters_immutable;
DROP TRIGGER marking_jobs_hybrid_parameters_insert;
DROP TRIGGER marking_answer_detections_hybrid_insert;
ALTER TABLE marking_answer_detections DROP COLUMN review_reason;
ALTER TABLE marking_answer_detections DROP COLUMN automatic_state;
ALTER TABLE marking_answer_detections DROP COLUMN color_signal;
ALTER TABLE marking_answer_detections DROP COLUMN grayscale_signal;
ALTER TABLE marking_answer_detections DROP COLUMN chroma_ratio;
ALTER TABLE marking_answer_detections DROP COLUMN dark_ratio;
ALTER TABLE marking_answer_detections DROP COLUMN v2_state;
ALTER TABLE marking_answer_detections DROP COLUMN historical_state;
ALTER TABLE marking_jobs DROP COLUMN v2_chroma_ratio_threshold;
ALTER TABLE marking_jobs DROP COLUMN v2_chroma_pixel_threshold;
ALTER TABLE marking_jobs DROP COLUMN v2_dark_ratio_threshold;
ALTER TABLE marking_jobs DROP COLUMN v2_dark_pixel_threshold;
ALTER TABLE marking_jobs DROP COLUMN v2_roi_radius_ratio;
ALTER TABLE marking_jobs DROP COLUMN review_policy_version;
