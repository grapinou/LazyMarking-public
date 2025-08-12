-- name: CreateQCM :exec
INSERT INTO
    qcm (name, user_id)
VALUES
    (:name, :user_id);

-- name: DeleteQCM :exec
DELETE FROM
   qcm 
WHERE
    id = :id
    AND user_id = :user_id;

-- name: GetAllQCM :many
SELECT
    *
FROM
   qcm 
WHERE
    user_id = :user_id
ORDER BY
    name;

-- name: UpdateQCM :exec
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