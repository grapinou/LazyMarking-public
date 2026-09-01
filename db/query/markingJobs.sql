-- name: CreateMarkingJob :one
INSERT INTO marking_jobs (
    user_id,
    exam_generated_id,
    result_schema_version,
    marking_algorithm_version,
    detection_threshold,
    ambiguity_delta
)
SELECT
    :user_id,
    :exam_generated_id,
    :result_schema_version,
    :marking_algorithm_version,
    :detection_threshold,
    :ambiguity_delta
FROM exams_generated
WHERE id = :exam_generated_id
  AND user_id = :user_id
  AND status = 'success'
RETURNING id;

-- name: ValidateMarkingJobStudentExam :one
SELECT se.id
FROM marking_jobs AS mj
JOIN student_exam AS se
  ON se.exam_generated_id = mj.exam_generated_id
 AND se.user_id = mj.user_id
WHERE mj.id = :marking_job_id
  AND mj.user_id = :user_id
  AND se.id = :student_exam_id;

-- name: DeleteMarkingJob :execrows
DELETE FROM
    marking_jobs
WHERE
    id = :id
    AND user_id = :user_id;

-- name: DeleteFailedMarkingJob :execrows
DELETE FROM
    marking_jobs
WHERE
    id = :id
    AND user_id = :user_id
    AND status = 'failed';

-- name: UpdateMarkingJobTotalPages :execrows
UPDATE
    marking_jobs
SET
    total_pages = :total_pages
WHERE
    id = :id
    AND user_id = :user_id
    AND status = 'running';

-- name: UpdateMarkingJobTotalExam :execrows
UPDATE
    marking_jobs
SET
    total_exams = :total_exams
WHERE
    id = :id
    AND user_id = :user_id
    AND status = 'running';

-- name: UpdateMarkingJobPageDone :execrows
UPDATE
    marking_jobs
SET
    done_pages = done_pages + 1
WHERE
    id = :id
    AND user_id = :user_id
    AND status = 'running'
    AND (total_pages IS NULL OR done_pages < total_pages);

-- name: UpdateMarkingJobExamDone :execrows
UPDATE
    marking_jobs
SET
    done_exams = done_exams + 1
WHERE
    id = :id
    AND user_id = :user_id
    AND status = 'running'
    AND (total_exams IS NULL OR done_exams < total_exams);

-- name: FailMarkingJob :execrows
UPDATE
    marking_jobs
SET
    status = 'failed',
    completed_at = CURRENT_TIMESTAMP
WHERE
    id = :id
    AND user_id = :user_id
    AND status = 'running';

-- name: GetMarkingStatus :one
SELECT
    status,
    status_pdf
FROM
    marking_jobs
WHERE
    id = :id
    AND user_id = :user_id;

-- name: GetMarkingProgress :one
SELECT
    total_pages,
    done_pages,
    total_exams,
    done_exams
FROM
    marking_jobs
WHERE
    id = :id
    AND user_id = :user_id;

-- name: CompleteMarkingJob :execrows
UPDATE
    marking_jobs
SET
    status = 'success',
    status_pdf = 'success',
    exam_name = :exam_name,
    mark_table_name = :mark_table_name,
    completed_at = CURRENT_TIMESTAMP
WHERE
    id = :id
    AND user_id = :user_id
    AND status = 'running';

-- name: CompleteMarkingJobWithResults :execrows
UPDATE marking_jobs
SET
    status = 'success',
    status_pdf = 'success',
    exam_name = :exam_name,
    mark_table_name = :mark_table_name,
    completed_at = CURRENT_TIMESTAMP
WHERE marking_jobs.id = sqlc.arg(id)
  AND marking_jobs.user_id = sqlc.arg(user_id)
  AND marking_jobs.status = 'running'
  AND marking_jobs.result_schema_version = sqlc.arg(result_schema_version)
  AND marking_jobs.marking_algorithm_version = sqlc.arg(marking_algorithm_version)
  AND marking_jobs.detection_threshold = sqlc.arg(detection_threshold)
  AND NOT EXISTS (
      SELECT 1
      FROM student_exam AS se
      WHERE se.exam_generated_id = marking_jobs.exam_generated_id
        AND se.user_id = marking_jobs.user_id
        AND NOT EXISTS (
            SELECT 1
            FROM marking_copy_results AS mcr
            WHERE mcr.marking_job_id = marking_jobs.id
              AND mcr.student_exam_id = se.id
              AND mcr.user_id = marking_jobs.user_id
        )
  )
  AND NOT EXISTS (
      SELECT 1
      FROM marking_copy_results AS mcr
      LEFT JOIN student_exam_content AS sec
        ON sec.student_exam_id = mcr.student_exam_id
       AND sec.user_id = mcr.user_id
      WHERE mcr.marking_job_id = marking_jobs.id
        AND mcr.outcome = 'corrected'
        AND (
            sec.student_exam_id IS NULL
            OR json_array_length(sec.content, '$.questions') < 1
            OR (SELECT COUNT(*) FROM marking_question_results AS mqr
                WHERE mqr.copy_result_id = mcr.id)
               != json_array_length(sec.content, '$.questions')
            OR EXISTS (
                SELECT 1
                FROM json_each(sec.content, '$.questions') AS snapshot_question
                LEFT JOIN marking_question_results AS mqr
                  ON mqr.copy_result_id = mcr.id
                 AND mqr.question_index = CAST(snapshot_question.key AS INTEGER)
                WHERE mqr.id IS NULL
                   OR (SELECT COUNT(*) FROM marking_answer_detections AS mad
                       WHERE mad.question_result_id = mqr.id)
                      != json_array_length(snapshot_question.value, '$.answers')
            )
            OR (SELECT COUNT(*)
                FROM marking_aligned_pages AS map
                WHERE map.copy_result_id = mcr.id)
               != mcr.expected_pages
            OR EXISTS (
                SELECT 1
                FROM student_exam_page_content AS sep
                WHERE sep.student_exam_id = mcr.student_exam_id
                  AND sep.user_id = mcr.user_id
                  AND NOT EXISTS (
                      SELECT 1
                      FROM marking_aligned_pages AS map
                      WHERE map.copy_result_id = mcr.id
                        AND map.page_exam = sep.page
                  )
            )
        )
  );

-- name: GetExamAndMarkName :one
SELECT
    exam_name,
    mark_table_name
FROM
    marking_jobs
WHERE
    id = :id
    AND user_id = :user_id
    AND status = 'success'
    AND status_pdf = 'success'
    AND exam_name IS NOT NULL
    AND mark_table_name IS NOT NULL;

-- name: ListRunningMarkingJobs :many
SELECT
    mj.id,
    mj.user_id,
    u.username
FROM marking_jobs AS mj
JOIN users AS u ON u.id = mj.user_id
WHERE mj.status = 'running';

-- name: ListExpiredFailedMarkingJobs :many
SELECT
    mj.id,
    mj.user_id,
    u.username
FROM marking_jobs AS mj
JOIN users AS u ON u.id = mj.user_id
WHERE mj.status = 'failed'
  AND mj.completed_at IS NOT NULL
  AND unixepoch(mj.completed_at) < unixepoch(:cutoff);
