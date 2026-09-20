-- name: ListMarkingJobHistory :many
SELECT
    mj.id,
    mj.exam_generated_id,
    e.name AS exam_name,
    cc.name AS class_code_name,
    mj.source_pdf_filename,
    mj.status,
    mj.status_pdf,
    mj.completed_at,
    COUNT(DISTINCT mcr.id) FILTER (WHERE mcr.outcome = 'corrected') AS corrected_copies,
    COUNT(DISTINCT mcr.id) FILTER (WHERE mcr.outcome IN ('incomplete', 'error')) AS issue_copies,
    COUNT(DISTINCT mcr.id) FILTER (WHERE mcr.outcome = 'not_seen') AS not_seen_copies,
    CAST(COALESCE(SUM(CASE WHEN
        mcr.outcome = 'corrected'
        AND mad.id IS NOT NULL
        AND mar.id IS NULL
        AND (
            (mj.review_policy_version IN ('detector-agreement-v1', 'detector-color-confidence-v1')
             AND mad.review_reason = 'detector_disagreement')
            OR
            (mj.review_policy_version IS NULL
             AND mj.detection_threshold IS NOT NULL
             AND mj.ambiguity_delta IS NOT NULL
             AND ABS(mad.mean_gray - mj.detection_threshold) <= mj.ambiguity_delta)
        )
        THEN 1 ELSE 0 END), 0) AS INTEGER) AS pending_reviews
FROM marking_jobs AS mj
JOIN exams_generated AS eg
  ON eg.id = mj.exam_generated_id
 AND eg.user_id = mj.user_id
JOIN exams AS e
  ON e.id = eg.exam_id
 AND e.user_id = mj.user_id
JOIN class_codes AS cc
  ON cc.id = e.class_code_id
 AND cc.user_id = mj.user_id
LEFT JOIN marking_copy_results AS mcr
  ON mcr.marking_job_id = mj.id
 AND mcr.user_id = mj.user_id
LEFT JOIN marking_question_results AS mqr ON mqr.copy_result_id = mcr.id
LEFT JOIN marking_answer_detections AS mad ON mad.question_result_id = mqr.id
LEFT JOIN marking_answer_reviews AS mar ON mar.answer_detection_id = mad.id
WHERE mj.user_id = sqlc.arg(user_id)
  AND (sqlc.arg(generation_id) = 0 OR mj.exam_generated_id = sqlc.arg(generation_id))
GROUP BY mj.id
ORDER BY mj.id DESC
LIMIT CASE WHEN sqlc.arg(generation_id) = 0 THEN 20 ELSE -1 END;

-- name: ListCurrentExamResultsForGeneration :many
WITH selected_generation AS (
    SELECT anchor.id AS exam_generated_id
    FROM exams_generated AS anchor
    WHERE anchor.id = sqlc.arg(generation_id)
      AND anchor.user_id = sqlc.arg(user_id)
),
ranked_results AS (
    SELECT
        mcr.id AS copy_result_id,
        mcr.student_exam_id,
        mcr.marking_job_id,
        mcr.outcome,
        mcr.score_half_units,
        mcr.total_points,
        mcr.completed_at,
        ROW_NUMBER() OVER (
            PARTITION BY mcr.student_exam_id
            ORDER BY
                CASE WHEN mcr.outcome = 'corrected' THEN 0 ELSE 1 END,
                mj.id DESC,
                mcr.id DESC
        ) AS result_rank
    FROM marking_copy_results AS mcr
    JOIN marking_jobs AS mj
      ON mj.id = mcr.marking_job_id
     AND mj.user_id = mcr.user_id
    JOIN selected_generation AS selected
      ON selected.exam_generated_id = mj.exam_generated_id
    WHERE mcr.user_id = sqlc.arg(user_id)
      AND mj.status = 'success'
      AND mj.status_pdf = 'success'
)
SELECT
    ranked.copy_result_id,
    ranked.student_exam_id,
    ranked.marking_job_id,
    s.first_name,
    s.last_name,
    ranked.outcome,
    ranked.score_half_units,
    ranked.total_points,
    ranked.completed_at,
    CAST(COALESCE((
        SELECT sec.content FROM student_exam_content AS sec
        WHERE sec.student_exam_id = ranked.student_exam_id
          AND sec.user_id = sqlc.arg(user_id)
    ), '') AS TEXT) AS snapshot_content,
    CAST((
        SELECT json_group_array(json_object(
            'question_index', mqr.question_index,
            'state', mqr.state,
            'score_half_units', mqr.score_half_units,
            'total_points', mqr.total_points
        ))
        FROM marking_question_results AS mqr
        WHERE mqr.copy_result_id = ranked.copy_result_id
    ) AS TEXT) AS question_results,
    CAST(COALESCE((
        SELECT COUNT(*)
        FROM marking_question_results AS mqr
        JOIN marking_answer_detections AS mad ON mad.question_result_id = mqr.id
        JOIN marking_jobs AS result_job ON result_job.id = ranked.marking_job_id
        LEFT JOIN marking_answer_reviews AS mar ON mar.answer_detection_id = mad.id
        WHERE mqr.copy_result_id = ranked.copy_result_id
          AND mar.id IS NULL
          AND (
              (result_job.review_policy_version IN ('detector-agreement-v1', 'detector-color-confidence-v1')
               AND mad.review_reason = 'detector_disagreement')
              OR
              (result_job.review_policy_version IS NULL
               AND result_job.detection_threshold IS NOT NULL
               AND result_job.ambiguity_delta IS NOT NULL
               AND ABS(mad.mean_gray - result_job.detection_threshold) <= result_job.ambiguity_delta)
          )
    ), 0) AS INTEGER) AS pending_reviews
FROM ranked_results AS ranked
JOIN student_exam AS se
  ON se.id = ranked.student_exam_id
 AND se.user_id = sqlc.arg(user_id)
JOIN students AS s
  ON s.id = se.student_id
 AND s.user_id = sqlc.arg(user_id)
-- Presentation order is applied once by buildMarkingExamSummary using French
-- collation, shared by HTML and PDF (SQLite NOCASE only handles ASCII).
WHERE ranked.result_rank = 1;

-- name: GetMarkingJobGeneration :one
SELECT exam_generated_id FROM marking_jobs
WHERE id = sqlc.arg(marking_job_id) AND user_id = sqlc.arg(user_id);

-- name: GetMarkingGeneration :one
SELECT eg.id, e.name AS exam_name, cc.name AS class_name
FROM exams_generated AS eg
JOIN exams AS e ON e.id = eg.exam_id AND e.user_id = eg.user_id
JOIN class_codes AS cc ON cc.id = e.class_code_id AND cc.user_id = eg.user_id
WHERE eg.id = sqlc.arg(generation_id) AND eg.user_id = sqlc.arg(user_id)
  AND eg.status = 'success';
