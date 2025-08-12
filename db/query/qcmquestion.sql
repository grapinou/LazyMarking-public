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