-- name: GetAllQuestionsByQCMID :many
SELECT
    questions.id         AS question_id,
    questions.content    AS question_content,
    qcm_questions.id     AS qcm_question_id
FROM questions
JOIN qcm_questions 
    ON qcm_questions.question_id = questions.id
WHERE questions.user_id = :user_id
  AND qcm_questions.qcm_id = :qcm_id;


-- name: CreateQCMQuestion :exec
INSERT INTO qcm_questions (qcm_id, question_id, user_id)
VALUES (:qcm_id, :question_id, :user_id);


-- name: GetQCMQuestionsIDs :many
SELECT question_id
FROM qcm_questions
WHERE user_id = :user_id
  AND qcm_id = :qcm_id
ORDER BY question_id;


-- name: GetQuestionContentByQCMQuestionID :one
SELECT
    questions.content
FROM questions
JOIN qcm_questions
    ON questions.id = qcm_questions.question_id
WHERE qcm_questions.user_id = :user_id
  AND qcm_questions.id = :qcm_question_id;


-- name: DeleteQCMQuestion :exec
DELETE FROM
   qcm_questions 
WHERE
    id = :id
    AND user_id = :user_id;