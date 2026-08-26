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

-- name: ListRunningExamGenerations :many
SELECT
    exams_generated.id,
    exams_generated.user_id,
    users.username
FROM
    exams_generated
    JOIN users ON users.id = exams_generated.user_id
WHERE
    exams_generated.status = 'running'
ORDER BY
    exams_generated.id;

-- name: UpdateExamGenerated :execrows
UPDATE
    exams_generated
SET
    status = :status
WHERE
    id = :id
    AND user_id = :user_id;

-- name: UpdateExamGeneratedProcessedStudent :execrows
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

-- name: GetExamsGeneratedSuccess :many
SELECT
    exams.name AS exam_name,
    class_codes.name AS class_code_name,
    strftime('%Y-%m-%d %H:%M', exams_generated.created_at) AS created_at
FROM
    exams_generated
    JOIN exams ON exams_generated.exam_id = exams.id
    JOIN class_codes ON exams.class_code_id = class_codes.id
WHERE
    exams_generated.status = 'success'
    AND exams_generated.user_id = :user_id
    AND exams.user_id = :user_id
    AND class_codes.user_id = :user_id
ORDER BY
    exams_generated.created_at DESC;
