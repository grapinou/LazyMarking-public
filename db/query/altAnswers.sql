-- name: CreateAltAnswer :exec
INSERT INTO
    alt_answers (alt_question_id, state, content, user_id)
VALUES
    (:alt_question_id, :state, :content, :user_id);

-- name: DeleteAltAnswer :exec
DELETE FROM
    alt_answers
WHERE
    id = :id
    AND user_id = :user_id;

-- name: GetAllAltAnswersByAltQuestionID :many
SELECT
    *
FROM
    alt_answers
WHERE
    alt_question_id = :alt_question_id
    AND user_id = :user_id;

-- name: UpdateAltAnswer :exec
UPDATE
    alt_answers
SET
    state = :state,
    content = :content
WHERE
    id = :id
    AND user_id = :user_id;

-- name: GetAltAnswerByID :one
SELECT
    *
FROM
    alt_answers
WHERE
    id = :id
    AND user_id = :user_id;


-- name: CountAltAnswerByAltQuestionID :one
SELECT COUNT(id)
FROM alt_answers
WHERE alt_question_id = :alt_question_id AND user_id = :user_id;
