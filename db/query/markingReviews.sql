-- name: CreateMarkingAnswerReview :one
INSERT INTO marking_answer_reviews (
    answer_detection_id,
    reviewer_user_id,
    reviewed_state
)
SELECT
    mad.id,
    sqlc.arg(reviewer_user_id),
    sqlc.arg(reviewed_state)
FROM marking_answer_detections AS mad
JOIN marking_question_results AS mqr ON mqr.id = mad.question_result_id
JOIN marking_copy_results AS mcr ON mcr.id = mqr.copy_result_id
JOIN marking_jobs AS mj
  ON mj.id = mcr.marking_job_id
 AND mj.user_id = mcr.user_id
WHERE mad.id = sqlc.arg(answer_detection_id)
  AND mj.user_id = sqlc.arg(reviewer_user_id)
RETURNING id;

-- name: UpdateMarkingAnswerReview :execrows
UPDATE marking_answer_reviews
SET
    reviewed_state = sqlc.arg(reviewed_state),
    reviewed_at = CURRENT_TIMESTAMP,
    revision = revision + 1
WHERE answer_detection_id = sqlc.arg(answer_detection_id)
  AND reviewer_user_id = sqlc.arg(reviewer_user_id)
  AND revision = sqlc.arg(expected_revision)
  AND EXISTS (
      SELECT 1
      FROM marking_answer_detections AS mad
      JOIN marking_question_results AS mqr ON mqr.id = mad.question_result_id
      JOIN marking_copy_results AS mcr ON mcr.id = mqr.copy_result_id
      JOIN marking_jobs AS mj
        ON mj.id = mcr.marking_job_id
       AND mj.user_id = mcr.user_id
      WHERE mad.id = marking_answer_reviews.answer_detection_id
        AND mj.user_id = sqlc.arg(reviewer_user_id)
  );

-- name: GetMarkingAnswerReview :one
SELECT mar.*
FROM marking_answer_reviews AS mar
JOIN marking_answer_detections AS mad ON mad.id = mar.answer_detection_id
JOIN marking_question_results AS mqr ON mqr.id = mad.question_result_id
JOIN marking_copy_results AS mcr ON mcr.id = mqr.copy_result_id
JOIN marking_jobs AS mj
  ON mj.id = mcr.marking_job_id
 AND mj.user_id = mcr.user_id
WHERE mar.answer_detection_id = sqlc.arg(answer_detection_id)
  AND mj.user_id = sqlc.arg(user_id);

-- name: GetEffectiveAnswerDetection :one
SELECT
    mad.id,
    mad.detected_state,
    mad.mean_gray,
    mar.reviewed_state AS "reviewed_state?",
    COALESCE(mar.reviewed_state, mad.detected_state) AS effective_state,
    mar.reviewed_at,
    mar.revision
FROM marking_answer_detections AS mad
JOIN marking_question_results AS mqr ON mqr.id = mad.question_result_id
JOIN marking_copy_results AS mcr ON mcr.id = mqr.copy_result_id
JOIN marking_jobs AS mj
  ON mj.id = mcr.marking_job_id
 AND mj.user_id = mcr.user_id
LEFT JOIN marking_answer_reviews AS mar ON mar.answer_detection_id = mad.id
WHERE mad.id = sqlc.arg(answer_detection_id)
  AND mj.user_id = sqlc.arg(user_id);

-- name: ListMarkingReviewCandidates :many
SELECT
    mad.id AS answer_detection_id,
    mcr.id AS copy_result_id,
    mcr.student_exam_id,
    mqr.question_index,
    mad.answer_index,
    mad.detected_state,
    mar.reviewed_state AS "reviewed_state?",
    COALESCE(mar.reviewed_state, mad.detected_state) AS effective_state,
    mad.mean_gray,
    mj.detection_threshold AS threshold,
    mj.ambiguity_delta,
    map.id AS aligned_page_id,
    map.page_exam,
    map.storage_key
FROM marking_jobs AS mj
JOIN marking_copy_results AS mcr
  ON mcr.marking_job_id = mj.id
 AND mcr.user_id = mj.user_id
 AND mcr.outcome = 'corrected'
JOIN marking_question_results AS mqr ON mqr.copy_result_id = mcr.id
JOIN marking_answer_detections AS mad ON mad.question_result_id = mqr.id
JOIN student_exam_page_content AS sep
  ON sep.student_exam_id = mcr.student_exam_id
 AND sep.user_id = mcr.user_id
 AND mqr.question_index >= (
     SELECT COALESCE(SUM(json_array_length(previous.content, '$.questions')), 0)
     FROM student_exam_page_content AS previous
     WHERE previous.student_exam_id = sep.student_exam_id
       AND previous.user_id = sep.user_id
       AND previous.page < sep.page
 )
 AND mqr.question_index < (
     SELECT COALESCE(SUM(json_array_length(previous.content, '$.questions')), 0)
     FROM student_exam_page_content AS previous
     WHERE previous.student_exam_id = sep.student_exam_id
       AND previous.user_id = sep.user_id
       AND previous.page < sep.page
 ) + json_array_length(sep.content, '$.questions')
JOIN marking_aligned_pages AS map
  ON map.copy_result_id = mcr.id
 AND map.user_id = mcr.user_id
 AND map.page_exam = sep.page
LEFT JOIN marking_answer_reviews AS mar ON mar.answer_detection_id = mad.id
WHERE mj.id = sqlc.arg(marking_job_id)
  AND mj.user_id = sqlc.arg(user_id)
  AND mj.status = 'success'
  AND mj.detection_threshold IS NOT NULL
  AND mj.ambiguity_delta IS NOT NULL
  AND ABS(mad.mean_gray - mj.detection_threshold) <= mj.ambiguity_delta
ORDER BY mcr.student_exam_id, mqr.question_index, mad.answer_index;

-- name: ListPendingMarkingReviewCandidates :many
SELECT
    mad.id AS answer_detection_id,
    mcr.id AS copy_result_id,
    mcr.student_exam_id,
    mqr.question_index,
    mad.answer_index,
    mad.detected_state,
    mad.mean_gray,
    mj.detection_threshold AS threshold,
    mj.ambiguity_delta,
    map.id AS aligned_page_id,
    map.page_exam,
    map.storage_key
FROM marking_jobs AS mj
JOIN marking_copy_results AS mcr
  ON mcr.marking_job_id = mj.id
 AND mcr.user_id = mj.user_id
 AND mcr.outcome = 'corrected'
JOIN marking_question_results AS mqr ON mqr.copy_result_id = mcr.id
JOIN marking_answer_detections AS mad ON mad.question_result_id = mqr.id
LEFT JOIN marking_answer_reviews AS mar ON mar.answer_detection_id = mad.id
JOIN student_exam_page_content AS sep
  ON sep.student_exam_id = mcr.student_exam_id
 AND sep.user_id = mcr.user_id
 AND mqr.question_index >= (
     SELECT COALESCE(SUM(json_array_length(previous.content, '$.questions')), 0)
     FROM student_exam_page_content AS previous
     WHERE previous.student_exam_id = sep.student_exam_id
       AND previous.user_id = sep.user_id
       AND previous.page < sep.page
 )
 AND mqr.question_index < (
     SELECT COALESCE(SUM(json_array_length(previous.content, '$.questions')), 0)
     FROM student_exam_page_content AS previous
     WHERE previous.student_exam_id = sep.student_exam_id
       AND previous.user_id = sep.user_id
       AND previous.page < sep.page
 ) + json_array_length(sep.content, '$.questions')
JOIN marking_aligned_pages AS map
  ON map.copy_result_id = mcr.id
 AND map.user_id = mcr.user_id
 AND map.page_exam = sep.page
WHERE mj.id = sqlc.arg(marking_job_id)
  AND mj.user_id = sqlc.arg(user_id)
  AND mj.status = 'success'
  AND mj.detection_threshold IS NOT NULL
  AND mj.ambiguity_delta IS NOT NULL
  AND ABS(mad.mean_gray - mj.detection_threshold) <= mj.ambiguity_delta
  AND mar.id IS NULL
ORDER BY mcr.student_exam_id, mqr.question_index, mad.answer_index;

-- name: GetMarkingReviewSummary :one
SELECT
    mj.ambiguity_delta,
    COUNT(mad.id) AS total_candidates,
    COUNT(mar.id) AS reviewed_candidates,
    COUNT(mad.id) - COUNT(mar.id) AS pending_candidates
FROM marking_jobs AS mj
LEFT JOIN marking_copy_results AS mcr
  ON mcr.marking_job_id = mj.id
 AND mcr.user_id = mj.user_id
 AND mcr.outcome = 'corrected'
LEFT JOIN marking_question_results AS mqr ON mqr.copy_result_id = mcr.id
LEFT JOIN marking_answer_detections AS mad
  ON mad.question_result_id = mqr.id
 AND mj.detection_threshold IS NOT NULL
 AND mj.ambiguity_delta IS NOT NULL
 AND ABS(mad.mean_gray - mj.detection_threshold) <= mj.ambiguity_delta
LEFT JOIN marking_answer_reviews AS mar ON mar.answer_detection_id = mad.id
WHERE mj.id = sqlc.arg(marking_job_id)
  AND mj.user_id = sqlc.arg(user_id)
  AND mj.status = 'success'
GROUP BY mj.id, mj.ambiguity_delta;

-- name: GetMarkingAnswerReviewTarget :one
SELECT
    mj.review_revision AS job_review_revision,
    mj.artifacts_revision,
    mcr.id AS copy_result_id,
    mcr.student_exam_id,
    mqr.id AS question_result_id,
    mqr.question_index,
    mqr.total_points,
    mad.detected_state,
    mar.reviewed_state AS "reviewed_state?",
    COALESCE(mar.reviewed_state, mad.detected_state) AS effective_state,
    mar.revision AS "answer_review_revision?",
    sec.content AS snapshot_content
FROM marking_jobs AS mj
JOIN marking_copy_results AS mcr
  ON mcr.marking_job_id = mj.id
 AND mcr.user_id = mj.user_id
 AND mcr.outcome = 'corrected'
JOIN marking_question_results AS mqr ON mqr.copy_result_id = mcr.id
JOIN marking_answer_detections AS mad ON mad.question_result_id = mqr.id
JOIN student_exam_content AS sec
  ON sec.student_exam_id = mcr.student_exam_id
 AND sec.user_id = mcr.user_id
LEFT JOIN marking_answer_reviews AS mar ON mar.answer_detection_id = mad.id
WHERE mj.id = sqlc.arg(marking_job_id)
  AND mj.user_id = sqlc.arg(user_id)
  AND mj.status = 'success'
  AND mad.id = sqlc.arg(answer_detection_id);

-- name: ListEffectiveQuestionAnswersForReview :many
SELECT
    mad.answer_index,
    mad.detected_state,
    mar.reviewed_state AS "reviewed_state?",
    COALESCE(mar.reviewed_state, mad.detected_state) AS effective_state
FROM marking_answer_detections AS mad
JOIN marking_question_results AS mqr ON mqr.id = mad.question_result_id
JOIN marking_copy_results AS mcr ON mcr.id = mqr.copy_result_id
JOIN marking_jobs AS mj
  ON mj.id = mcr.marking_job_id
 AND mj.user_id = mcr.user_id
LEFT JOIN marking_answer_reviews AS mar ON mar.answer_detection_id = mad.id
WHERE mad.question_result_id = sqlc.arg(question_result_id)
  AND mcr.id = sqlc.arg(copy_result_id)
  AND mj.id = sqlc.arg(marking_job_id)
  AND mj.user_id = sqlc.arg(user_id)
  AND mj.status = 'success'
  AND mcr.outcome = 'corrected'
ORDER BY mad.answer_index;

-- name: UpdateMarkingQuestionResultFromReview :execrows
UPDATE marking_question_results
SET
    state = sqlc.arg(state),
    score_half_units = sqlc.arg(score_half_units)
WHERE marking_question_results.id = sqlc.arg(question_result_id)
  AND marking_question_results.copy_result_id = sqlc.arg(copy_result_id)
  AND EXISTS (
      SELECT 1
      FROM marking_copy_results AS mcr
      JOIN marking_jobs AS mj
        ON mj.id = mcr.marking_job_id
       AND mj.user_id = mcr.user_id
      WHERE mcr.id = marking_question_results.copy_result_id
        AND mcr.user_id = sqlc.arg(user_id)
        AND mcr.outcome = 'corrected'
        AND mj.id = sqlc.arg(marking_job_id)
        AND mj.status = 'success'
  );

-- name: RecalculateMarkingCopyScoreFromQuestions :execrows
UPDATE marking_copy_results
SET score_half_units = (
    SELECT COALESCE(SUM(mqr.score_half_units), 0)
    FROM marking_question_results AS mqr
    WHERE mqr.copy_result_id = marking_copy_results.id
)
WHERE marking_copy_results.id = sqlc.arg(copy_result_id)
  AND marking_copy_results.user_id = sqlc.arg(user_id)
  AND marking_copy_results.marking_job_id = sqlc.arg(marking_job_id)
  AND marking_copy_results.outcome = 'corrected'
  AND EXISTS (
      SELECT 1
      FROM marking_jobs AS mj
      WHERE mj.id = marking_copy_results.marking_job_id
        AND mj.user_id = marking_copy_results.user_id
        AND mj.status = 'success'
  );

-- name: AdvanceMarkingJobReviewRevision :execrows
UPDATE marking_jobs
SET
    review_revision = review_revision + 1,
    artifacts_revision = CASE
        WHEN CAST(sqlc.arg(effective_changed) AS INTEGER) = 0
         AND artifacts_revision = review_revision
        THEN review_revision + 1
        ELSE artifacts_revision
    END
WHERE id = sqlc.arg(marking_job_id)
  AND user_id = sqlc.arg(user_id)
  AND status = 'success'
  AND review_revision = sqlc.arg(expected_review_revision);

-- name: CreateMarkingAlignedPage :one
INSERT INTO marking_aligned_pages (
    user_id,
    copy_result_id,
    page_exam,
    storage_key,
    width,
    height,
    sha256
)
SELECT
    sqlc.arg(user_id),
    mcr.id,
    sqlc.arg(page_exam),
    sqlc.arg(storage_key),
    sqlc.arg(width),
    sqlc.arg(height),
    sqlc.arg(sha256)
FROM marking_copy_results AS mcr
JOIN marking_jobs AS mj
  ON mj.id = mcr.marking_job_id
 AND mj.user_id = mcr.user_id
WHERE mcr.id = sqlc.arg(copy_result_id)
  AND mcr.user_id = sqlc.arg(user_id)
  AND mj.user_id = sqlc.arg(user_id)
  AND mcr.outcome = 'corrected'
  AND sqlc.arg(storage_key) = printf(
      'aligned/student-exam-%d/page-%d.png',
      mcr.student_exam_id,
      sqlc.arg(page_exam)
  )
  AND (SELECT COUNT(*)
       FROM student_exam_page_content AS sep
       WHERE sep.student_exam_id = mcr.student_exam_id
         AND sep.user_id = mcr.user_id
         AND sep.page = sqlc.arg(page_exam)) = 1
RETURNING id;

-- name: ValidateMarkingAlignedPageTarget :one
SELECT mcr.id
FROM marking_copy_results AS mcr
JOIN marking_jobs AS mj
  ON mj.id = mcr.marking_job_id
 AND mj.user_id = mcr.user_id
WHERE mcr.id = sqlc.arg(copy_result_id)
  AND mcr.marking_job_id = sqlc.arg(marking_job_id)
  AND mcr.student_exam_id = sqlc.arg(student_exam_id)
  AND mcr.user_id = sqlc.arg(user_id)
  AND mcr.outcome = 'corrected'
  AND (SELECT COUNT(*)
       FROM student_exam_page_content AS sep
       WHERE sep.student_exam_id = mcr.student_exam_id
         AND sep.user_id = mcr.user_id
         AND sep.page = sqlc.arg(page_exam)) = 1;

-- name: GetMarkingAlignedPage :one
SELECT
    map.id,
    map.user_id,
    map.copy_result_id,
    map.page_exam,
    map.storage_key,
    map.width,
    map.height,
    map.sha256,
    map.created_at,
    mcr.student_exam_id,
    mcr.marking_job_id,
    u.username
FROM marking_aligned_pages AS map
JOIN marking_copy_results AS mcr ON mcr.id = map.copy_result_id
JOIN marking_jobs AS mj
  ON mj.id = mcr.marking_job_id
 AND mj.user_id = mcr.user_id
JOIN users AS u ON u.id = mj.user_id
WHERE map.copy_result_id = sqlc.arg(copy_result_id)
  AND map.page_exam = sqlc.arg(page_exam)
  AND mj.user_id = sqlc.arg(user_id);
