-- name: CreateQCM :exec
INSERT INTO
    qcm (name, user_id)
VALUES
    (:name, :user_id);

-- name: DeleteQCM :execrows
DELETE FROM
   qcm 
WHERE
    id = :id
    AND user_id = :user_id;

-- name: GetAllQCM :many
SELECT
    qcm.id,
    qcm.name,
    qcm.user_id,
    COUNT(qcm_questions.id) AS question_count
FROM
   qcm
LEFT JOIN qcm_questions
    ON qcm_questions.qcm_id = qcm.id
    AND qcm_questions.user_id = qcm.user_id
WHERE
    qcm.user_id = :user_id
GROUP BY
    qcm.id,
    qcm.name,
    qcm.user_id
ORDER BY
    qcm.id DESC;

-- name: UpdateQCM :execrows
UPDATE
   qcm 
SET
    name = :name
WHERE
    id = :id
    AND user_id = :user_id;

-- name: GetQCMNameByID :one
SELECT
    name
FROM
   qcm 
WHERE
    id = :id
    AND user_id = :user_id;
