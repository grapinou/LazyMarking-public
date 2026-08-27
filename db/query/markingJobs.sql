-- name: CreateMarkingJob :one
INSERT INTO
    marking_jobs (user_id)
VALUES
    (:user_id) RETURNING id;

-- name: DeleteMarkingJob :execrows
DELETE FROM
    marking_jobs
WHERE
    id = :id
    AND user_id = :user_id;

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

-- name: ListExpiredMarkingJobs :many
SELECT
    mj.id,
    mj.user_id,
    u.username
FROM marking_jobs AS mj
JOIN users AS u ON u.id = mj.user_id
WHERE mj.status IN ('success', 'failed')
  AND mj.completed_at IS NOT NULL
  AND unixepoch(mj.completed_at) < unixepoch(:cutoff);
