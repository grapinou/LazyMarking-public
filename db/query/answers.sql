-- name: CreateAnswer :exec
INSERT INTO
    answers (question_id, state, content, user_id)
VALUES
    (:question_id, :state, :content, :user_id);

-- name: DeleteAnswer :exec
DELETE FROM
    answers
WHERE
    id = :id
    AND user_id = :user_id;

-- name: GetAllAnswersByQuestionID :many
SELECT
    *
FROM
    answers
WHERE
    question_id = :question_id
    AND user_id = :user_id;

-- name: UpdateAnswer :exec
UPDATE
    answers
SET
    state = :state,
    content = :content
WHERE
    id = :id
    AND user_id = :user_id;

-- name: GetAnswerByID :one
SELECT
    *
FROM
    answers
WHERE
    id = :id
    AND user_id = :user_id;


-- name: CountAnswerByQuestionID :one
SELECT COUNT(id)
FROM answers
WHERE question_id = :question_id AND user_id = :user_id;
