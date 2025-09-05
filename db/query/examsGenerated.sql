-- name: CreateExamGenerated :one
INSERT INTO
    exams_generated (
        exam_id,
        total_students,
        user_id
    )
VALUES
    (
        :exam_id,
        :total_students,
        :user_id
    ) RETURNING id;

-- name: DeleteExamGenerated :exec
DELETE FROM
    exams_generated
WHERE
    id = :id
    AND user_id = :user_id;

-- name: UpdateExamGenerated :exec
UPDATE
    exams_generated
SET
    status = :status
WHERE
    id = :id
    AND user_id = :user_id;

-- name: UpdateExamGeneratedProcessedStudent :exec
UPDATE
    exams_generated
SET
    processed_students = processed_students + 1
WHERE
    id = :id
    AND user_id = :user_id;

-- name: GetExamGeneratedProgress :one
SELECT
    processed_students,
    total_students
FROM
    exams_generated
WHERE
    id = :id
    AND user_id = :user_id;

-- name: GetExamStatus :one
SELECT
    status
FROM
    exams_generated
WHERE
    id = :id
    AND user_id = :user_id;