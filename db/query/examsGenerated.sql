-- name: CreateExamGenerated :one
INSERT INTO
    exams_generated (exam_id, user_id)
VALUES
    (:exam_id, :user_id) RETURNING id;

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