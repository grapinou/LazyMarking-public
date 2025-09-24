-- name: CreateMarkingJob :one
INSERT INTO
    marking_jobs (user_id)
VALUES
    (:user_id) RETURNING id;

-- name: DeleteMarkingJob :exec
DELETE FROM
    marking_jobs
WHERE
    id = :id
    AND user_id = :user_id;

-- name: UpdateMarkingJobTotalPages :exec
UPDATE
    marking_jobs
SET
    total_pages = :total_pages
WHERE
    id = :id
    AND user_id = :user_id;

-- name: UpdateMarkingJobTotalExam :exec
UPDATE
    marking_jobs
SET
    total_exams = :total_exams
WHERE
    id = :id
    AND user_id = :user_id;

-- name: UpdateMarkingJobPageDone :exec
UPDATE
    marking_jobs
SET
    done_pages = done_pages + 1
WHERE
    id = :id
    AND user_id = :user_id;

-- name: UpdateMarkingJobExamDone :exec
UPDATE
    marking_jobs
SET
    done_exams = done_exams + 1
WHERE
    id = :id
    AND user_id = :user_id;

-- name: UpdateMarkingJobStatus :exec
UPDATE
    marking_jobs
SET
    status = :status
WHERE
    id = :id
    AND user_id = :user_id;

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

-- name: UpdateMarkingJobStatusPDF :exec
UPDATE
    marking_jobs
SET
    status_pdf = :status_pdf,
    exam_name = :exam_name,
    mark_table_name = :mark_table_name
WHERE
    id = :id
    AND user_id = :user_id;

-- name: GetExamAndMarkName :one
SELECT
    exam_name,
    mark_table_name
FROM
    marking_jobs
WHERE
    id = :id
    AND user_id = :user_id;