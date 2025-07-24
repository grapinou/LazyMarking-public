-- name: GetAllAltQuestions :many
SELECT
    *
FROM
    alt_questions
WHERE
    question_id = :question_id
    AND user_id = :user_id;

-- name: CreateAltQuestion :exec
INSERT INTO
    alt_questions (
        question_id,
        content,
        user_id
    )
VALUES
    (
        :question_id,
        :content,
        :user_id
    );

-- name: GetAltQuestionByID :one
SELECT
    *
FROM
    alt_questions
WHERE
    id = :id
    AND user_id = :user_id;

-- name: UpdateAltQuestion :exec
UPDATE
    alt_questions
SET
    content = :content
WHERE
    id = :id
    AND user_id = :user_id;

-- name: DeleteAltQuestion :exec
DELETE FROM
    alt_questions
WHERE
    id = :id
    AND user_id = :user_id;