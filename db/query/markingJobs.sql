-- name: CreateMarkingJob :one
INSERT INTO
    marking_jobs (user_id, total_pages)
VALUES
    (:user_id, :total_pages) RETURNING id;

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