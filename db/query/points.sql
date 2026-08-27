-- name: CreatePoint :exec
INSERT INTO
    points (point_value, user_id)
VALUES
    (:point_value, :user_id);

-- name: DeletePoint :execrows
DELETE FROM
    points
WHERE
    id = :id
    AND user_id = :user_id;

-- name: GetAllPoints :many
SELECT
    *
FROM
    points
WHERE
    user_id = :user_id
ORDER BY
    point_value ASC;

-- name: UpdatePoint :execrows
UPDATE
    points
SET
    point_value = :point_value
WHERE
    id = :id
    AND user_id = :user_id;

-- name: GetPointByID :one
SELECT
    point_value
FROM
    points
WHERE
    id = :id
    AND user_id = :user_id;
