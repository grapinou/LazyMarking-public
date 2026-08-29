-- name: GetAllQuestionsByQCMID :many
SELECT
    questions.id         AS question_id,
    questions.content    AS question_content,
    qcm_questions.id     AS qcm_question_id,
    qcm_questions.position
FROM questions
JOIN qcm_questions 
    ON qcm_questions.question_id = questions.id
WHERE questions.user_id = :user_id
  AND qcm_questions.user_id = :user_id
  AND qcm_questions.qcm_id = :qcm_id
  AND EXISTS (
      SELECT 1 FROM qcm
      WHERE id = :qcm_id AND user_id = sqlc.arg(user_id)
  )
ORDER BY qcm_questions.position ASC;


-- name: CreateQCMQuestion :execrows
INSERT INTO qcm_questions (qcm_id, question_id, user_id, position)
SELECT
    :qcm_id,
    :question_id,
    :user_id,
    COALESCE((
        SELECT MAX(position)
        FROM qcm_questions
        WHERE qcm_id = :qcm_id
    ), 0) + 1
WHERE EXISTS (
    SELECT 1 FROM qcm
    WHERE id = :qcm_id AND user_id = sqlc.arg(user_id)
)
AND EXISTS (
    SELECT 1 FROM questions
    WHERE id = :question_id AND user_id = sqlc.arg(user_id)
);


-- name: GetQCMQuestionsIDs :many
SELECT question_id
FROM qcm_questions
WHERE qcm_questions.user_id = sqlc.arg(user_id)
  AND qcm_id = :qcm_id
  AND EXISTS (
      SELECT 1 FROM qcm
      WHERE id = :qcm_id AND user_id = sqlc.arg(user_id)
  )
ORDER BY position ASC;


-- name: GetQCMQuestionPosition :one
SELECT
    qcm_questions.position,
    (
        SELECT sibling.position
        FROM qcm_questions sibling
        WHERE sibling.qcm_id = :qcm_id
          AND sibling.user_id = sqlc.arg(user_id)
        ORDER BY sibling.position DESC
        LIMIT 1
    ) AS max_position
FROM qcm_questions
WHERE qcm_questions.id = sqlc.arg(id)
  AND qcm_questions.qcm_id = :qcm_id
  AND qcm_questions.user_id = sqlc.arg(user_id)
  AND EXISTS (
      SELECT 1 FROM qcm
      WHERE id = :qcm_id AND user_id = sqlc.arg(user_id)
  );


-- name: GetQCMQuestionByPosition :one
SELECT qcm_questions.id, qcm_questions.position
FROM qcm_questions
WHERE qcm_questions.qcm_id = :qcm_id
  AND qcm_questions.user_id = sqlc.arg(user_id)
  AND qcm_questions.position = :position
  AND EXISTS (
      SELECT 1 FROM qcm
      WHERE id = :qcm_id AND user_id = sqlc.arg(user_id)
  );


-- name: MoveQCMQuestionToPosition :execrows
UPDATE qcm_questions
SET position = sqlc.arg(position)
WHERE qcm_questions.id = sqlc.arg(id)
  AND qcm_questions.qcm_id = :qcm_id
  AND qcm_questions.user_id = sqlc.arg(user_id)
  AND EXISTS (
      SELECT 1 FROM qcm
      WHERE id = :qcm_id AND user_id = sqlc.arg(user_id)
  );


-- name: GetQuestionContentByQCMQuestionID :one
SELECT
    questions.content
FROM questions
JOIN qcm_questions
    ON questions.id = qcm_questions.question_id
WHERE qcm_questions.user_id = :user_id
  AND questions.user_id = :user_id
  AND qcm_questions.id = :qcm_question_id
  AND qcm_questions.qcm_id = :qcm_id
  AND EXISTS (
      SELECT 1 FROM qcm
      WHERE id = :qcm_id AND user_id = sqlc.arg(user_id)
  );


-- name: DeleteQCMQuestion :execrows
DELETE FROM
   qcm_questions 
WHERE
    qcm_questions.id = sqlc.arg(id)
    AND qcm_questions.qcm_id = :qcm_id
    AND qcm_questions.user_id = sqlc.arg(user_id)
    AND EXISTS (
        SELECT 1 FROM qcm
        WHERE id = :qcm_id AND user_id = sqlc.arg(user_id)
    );


-- name: MoveQCMQuestionPositionsToTemporaryRange :execrows
UPDATE qcm_questions
SET position = qcm_questions.position + sqlc.arg(max_position)
WHERE qcm_questions.qcm_id = :qcm_id
  AND qcm_questions.user_id = sqlc.arg(user_id)
  AND qcm_questions.position > sqlc.arg(deleted_position)
  AND EXISTS (
      SELECT 1 FROM qcm q
      WHERE q.id = :qcm_id AND q.user_id = sqlc.arg(user_id)
  );


-- name: CompactQCMQuestionPositions :execrows
UPDATE qcm_questions
SET position = qcm_questions.position - sqlc.arg(max_position) - 1
WHERE qcm_questions.qcm_id = :qcm_id
  AND qcm_questions.user_id = sqlc.arg(user_id)
  AND qcm_questions.position > sqlc.arg(max_position)
  AND EXISTS (
      SELECT 1 FROM qcm q
      WHERE q.id = :qcm_id AND q.user_id = sqlc.arg(user_id)
  );
