-- name: CreateMarkingCopyResult :one
INSERT INTO marking_copy_results (
    user_id,
    marking_job_id,
    student_exam_id,
    outcome,
    expected_pages,
    detected_pages,
    score_half_units,
    total_points,
    failure_code,
    failure_detail
)
SELECT
    sqlc.arg(user_id),
    sqlc.arg(marking_job_id),
    sqlc.arg(student_exam_id),
    sqlc.arg(outcome),
    sqlc.arg(expected_pages),
    sqlc.arg(detected_pages),
    sqlc.narg(score_half_units),
    sqlc.narg(total_points),
    sqlc.narg(failure_code),
    sqlc.narg(failure_detail)
FROM marking_jobs AS mj
JOIN student_exam AS se
  ON se.exam_generated_id = mj.exam_generated_id
 AND se.user_id = mj.user_id
WHERE mj.id = sqlc.arg(marking_job_id)
  AND mj.user_id = sqlc.arg(user_id)
  AND se.id = sqlc.arg(student_exam_id)
RETURNING id;

-- name: ListExpectedStudentExamsForMarkingJob :many
SELECT
    se.id AS student_exam_id,
    sec.page_tot AS expected_pages
FROM marking_jobs AS mj
JOIN student_exam AS se
  ON se.exam_generated_id = mj.exam_generated_id
 AND se.user_id = mj.user_id
JOIN student_exam_content AS sec
  ON sec.student_exam_id = se.id
 AND sec.user_id = se.user_id
WHERE mj.id = :marking_job_id
  AND mj.user_id = :user_id
ORDER BY se.id;

-- name: GetMarkingResultCoverage :one
SELECT
    (SELECT COUNT(*)
     FROM student_exam AS se
     WHERE se.exam_generated_id = mj.exam_generated_id
       AND se.user_id = mj.user_id) AS expected_count,
    (SELECT COUNT(*)
     FROM marking_copy_results AS mcr
     WHERE mcr.marking_job_id = mj.id
       AND mcr.user_id = mj.user_id) AS result_count
FROM marking_jobs AS mj
WHERE mj.id = :marking_job_id
  AND mj.user_id = :user_id;

-- name: CreateMarkingQuestionResult :one
INSERT INTO marking_question_results (
    copy_result_id,
    question_index,
    state,
    score_half_units,
    total_points
)
VALUES (
    :copy_result_id,
    :question_index,
    :state,
    :score_half_units,
    :total_points
)
RETURNING id;

-- name: CreateMarkingAnswerDetection :one
INSERT INTO marking_answer_detections (
    question_result_id,
    answer_index,
    detected_state,
    mean_gray
)
VALUES (
    :question_result_id,
    :answer_index,
    :detected_state,
    :mean_gray
)
RETURNING id;

-- name: CreateHybridMarkingAnswerDetection :one
INSERT INTO marking_answer_detections (
    question_result_id,
    answer_index,
    detected_state,
    mean_gray,
    historical_state,
    v2_state,
    dark_ratio,
    chroma_ratio,
    grayscale_signal,
    color_signal,
    automatic_state,
    review_reason
)
VALUES (
    :question_result_id,
    :answer_index,
    :detected_state,
    :mean_gray,
    :historical_state,
    :v2_state,
    :dark_ratio,
    :chroma_ratio,
    :grayscale_signal,
    :color_signal,
    :automatic_state,
    :review_reason
)
RETURNING id;

-- name: GetMarkingCopyResult :one
SELECT *
FROM marking_copy_results
WHERE id = :id AND user_id = :user_id;

-- name: ListMarkingQuestionResults :many
SELECT mqr.*
FROM marking_question_results AS mqr
JOIN marking_copy_results AS mcr ON mcr.id = mqr.copy_result_id
WHERE mqr.copy_result_id = :copy_result_id
  AND mcr.user_id = :user_id
ORDER BY mqr.question_index;

-- name: ListMarkingAnswerDetections :many
SELECT mad.*
FROM marking_answer_detections AS mad
JOIN marking_question_results AS mqr ON mqr.id = mad.question_result_id
JOIN marking_copy_results AS mcr ON mcr.id = mqr.copy_result_id
WHERE mad.question_result_id = :question_result_id
  AND mcr.user_id = :user_id
ORDER BY mad.answer_index;
