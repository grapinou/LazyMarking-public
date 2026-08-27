-- name: CreateAltAnswer :execrows
INSERT INTO
    alt_answers (alt_question_id, state, content, user_id)
SELECT :alt_question_id, :state, :content, :user_id
WHERE EXISTS (SELECT 1 FROM alt_questions a
              WHERE a.id = :alt_question_id AND a.question_id = :question_id AND a.user_id = :user_id);

-- name: DeleteAltAnswer :execrows
DELETE FROM
    alt_answers
WHERE
    alt_answers.id = :id
    AND alt_answers.alt_question_id = :alt_question_id
    AND alt_answers.user_id = :user_id
    AND EXISTS (SELECT 1 FROM alt_questions a
                WHERE a.id = alt_answers.alt_question_id AND a.question_id = :question_id AND a.user_id = :user_id);

-- name: GetAllAltAnswersByAltQuestionID :many
SELECT
    *
FROM
    alt_answers
WHERE
    alt_answers.alt_question_id = :alt_question_id
    AND alt_answers.user_id = :user_id
    AND EXISTS (SELECT 1 FROM alt_questions a WHERE a.id = alt_answers.alt_question_id AND a.user_id = :user_id);

-- name: UpdateAltAnswer :execrows
UPDATE
    alt_answers
SET
    state = :state,
    content = :content
WHERE
    alt_answers.id = :id
    AND alt_answers.alt_question_id = :alt_question_id
    AND alt_answers.user_id = :user_id
    AND EXISTS (SELECT 1 FROM alt_questions a
                WHERE a.id = alt_answers.alt_question_id AND a.question_id = :question_id AND a.user_id = :user_id);

-- name: GetAltAnswerByID :one
SELECT
    *
FROM
    alt_answers
WHERE
    alt_answers.id = :id
    AND alt_answers.alt_question_id = :alt_question_id
    AND alt_answers.user_id = :user_id
    AND EXISTS (SELECT 1 FROM alt_questions a
                WHERE a.id = alt_answers.alt_question_id AND a.question_id = :question_id AND a.user_id = :user_id);


-- name: CountAltAnswerByAltQuestionID :one
SELECT COUNT(id)
FROM alt_answers
WHERE alt_answers.alt_question_id = :alt_question_id AND alt_answers.user_id = :user_id
  AND EXISTS (SELECT 1 FROM alt_questions a WHERE a.id = alt_answers.alt_question_id AND a.user_id = :user_id);
