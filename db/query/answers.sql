-- name: CreateAnswer :execrows
INSERT INTO
    answers (question_id, state, content, user_id)
SELECT :question_id, :state, :content, :user_id
WHERE EXISTS (SELECT 1 FROM questions q WHERE q.id = :question_id AND q.user_id = :user_id);

-- name: DeleteAnswer :execrows
DELETE FROM
    answers
WHERE
    answers.id = :id
    AND answers.question_id = :question_id
    AND answers.user_id = :user_id
    AND EXISTS (SELECT 1 FROM questions q WHERE q.id = answers.question_id AND q.user_id = :user_id);

-- name: GetAllAnswersByQuestionID :many
SELECT
    *
FROM
    answers
WHERE
    answers.question_id = :question_id
    AND answers.user_id = :user_id
    AND EXISTS (SELECT 1 FROM questions q WHERE q.id = answers.question_id AND q.user_id = :user_id);

-- name: UpdateAnswer :execrows
UPDATE
    answers
SET
    state = :state,
    content = :content
WHERE
    answers.id = :id
    AND answers.question_id = :question_id
    AND answers.user_id = :user_id
    AND EXISTS (SELECT 1 FROM questions q WHERE q.id = answers.question_id AND q.user_id = :user_id);

-- name: GetAnswerByID :one
SELECT
    *
FROM
    answers
WHERE
    answers.id = :id
    AND answers.question_id = :question_id
    AND answers.user_id = :user_id
    AND EXISTS (SELECT 1 FROM questions q WHERE q.id = answers.question_id AND q.user_id = :user_id);


-- name: CountAnswerByQuestionID :one
SELECT COUNT(id)
FROM answers
WHERE answers.question_id = :question_id AND answers.user_id = :user_id
  AND EXISTS (SELECT 1 FROM questions q WHERE q.id = answers.question_id AND q.user_id = :user_id);
