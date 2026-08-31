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
    mar.reviewed_state,
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

-- name: GetMarkingAlignedPage :one
SELECT map.*
FROM marking_aligned_pages AS map
JOIN marking_copy_results AS mcr ON mcr.id = map.copy_result_id
JOIN marking_jobs AS mj
  ON mj.id = mcr.marking_job_id
 AND mj.user_id = mcr.user_id
WHERE map.copy_result_id = sqlc.arg(copy_result_id)
  AND map.page_exam = sqlc.arg(page_exam)
  AND mj.user_id = sqlc.arg(user_id);
