-- +goose Up
DROP TRIGGER marking_answer_detections_hybrid_insert;
DROP TRIGGER marking_jobs_hybrid_parameters_insert;

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
      AND (
        (EXISTS (
            SELECT 1
            FROM marking_question_results AS mqr
            JOIN marking_copy_results AS mcr ON mcr.id = mqr.copy_result_id
            JOIN marking_jobs AS mj ON mj.id = mcr.marking_job_id
            WHERE mqr.id = NEW.question_result_id
              AND mj.review_policy_version = 'detector-agreement-v1'
          )
          AND ((NEW.historical_state = NEW.v2_state
                AND NEW.automatic_state = NEW.historical_state
                AND NEW.review_reason IS NULL)
            OR (NEW.historical_state != NEW.v2_state
                AND NEW.automatic_state IS NULL
                AND NEW.review_reason = 'detector_disagreement')))
        OR
        (EXISTS (
            SELECT 1
            FROM marking_question_results AS mqr
            JOIN marking_copy_results AS mcr ON mcr.id = mqr.copy_result_id
            JOIN marking_jobs AS mj ON mj.id = mcr.marking_job_id
            WHERE mqr.id = NEW.question_result_id
              AND mj.review_policy_version = 'detector-color-confidence-v1'
          )
          AND ((NEW.historical_state = NEW.v2_state
                AND NEW.automatic_state = NEW.historical_state
                AND NEW.review_reason IS NULL)
            OR (NEW.historical_state = 0 AND NEW.v2_state = 1
                AND NEW.color_signal = 1
                AND NEW.automatic_state = 1
                AND NEW.review_reason IS NULL)
            OR (((NEW.historical_state = 0 AND NEW.v2_state = 1 AND NEW.color_signal = 0)
                  OR (NEW.historical_state = 1 AND NEW.v2_state = 0))
                AND NEW.automatic_state IS NULL
                AND NEW.review_reason = 'detector_disagreement')))
      ))
)
BEGIN
    SELECT RAISE(ABORT, 'hybrid answer detection metadata must match the job review policy');
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
    (NEW.review_policy_version IN ('detector-agreement-v1', 'detector-color-confidence-v1')
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

-- +goose Down
DROP TRIGGER marking_jobs_hybrid_parameters_insert;
DROP TRIGGER marking_answer_detections_hybrid_insert;

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
