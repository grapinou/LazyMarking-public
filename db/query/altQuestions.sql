-- name: GetAllAltQuestions :many
SELECT
    *
FROM
    alt_questions
WHERE
    alt_questions.question_id = :question_id
    AND alt_questions.user_id = :user_id
    AND EXISTS (SELECT 1 FROM questions q WHERE q.id = alt_questions.question_id AND q.user_id = :user_id);

-- name: GetAllOwnedAltQuestions :many
SELECT
    alt_questions.*
FROM
    alt_questions
WHERE
    alt_questions.user_id = :user_id
    AND EXISTS (
        SELECT 1
        FROM questions q
        WHERE q.id = alt_questions.question_id
          AND q.user_id = :user_id
    )
ORDER BY
    alt_questions.question_id,
    alt_questions.id;

-- name: CreateAltQuestion :execrows
INSERT INTO
    alt_questions (
        question_id,
        content,
        user_id
    )
SELECT :question_id, :content, :user_id
WHERE EXISTS (SELECT 1 FROM questions q WHERE q.id = :question_id AND q.user_id = :user_id);

-- name: GetAltQuestionByID :one
SELECT
    *
FROM
    alt_questions
WHERE
    alt_questions.id = :id
    AND alt_questions.user_id = :user_id
    AND EXISTS (SELECT 1 FROM questions q WHERE q.id = alt_questions.question_id AND q.user_id = :user_id);

-- name: GetAltQuestionByParentID :one
SELECT * FROM alt_questions
WHERE alt_questions.id = :id
  AND alt_questions.question_id = :question_id
  AND alt_questions.user_id = :user_id
  AND EXISTS (SELECT 1 FROM questions q WHERE q.id = alt_questions.question_id AND q.user_id = :user_id);

-- name: UpdateAltQuestion :execrows
UPDATE
    alt_questions
SET
    content = :content
WHERE
    alt_questions.id = :id
    AND alt_questions.question_id = :question_id
    AND alt_questions.user_id = :user_id
    AND EXISTS (SELECT 1 FROM questions q WHERE q.id = alt_questions.question_id AND q.user_id = :user_id);

-- name: DeleteAltQuestion :execrows
DELETE FROM
    alt_questions
WHERE
    alt_questions.id = :id
    AND alt_questions.question_id = :question_id
    AND alt_questions.user_id = :user_id
    AND EXISTS (SELECT 1 FROM questions q WHERE q.id = alt_questions.question_id AND q.user_id = :user_id);
