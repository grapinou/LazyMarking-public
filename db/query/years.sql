-- name: CreateYear :exec
INSERT INTO
    years (name, user_id)
VALUES
    (:name, :user_id);

-- name: DeleteYear :exec
DELETE FROM
    years
WHERE
    id = :id
    AND user_id = :user_id;

-- name: GetAllYears :many
SELECT
    *
FROM
    years
WHERE
    user_id = :user_id
ORDER BY
    name;

-- name: UpdateYear :exec
UPDATE
    years
SET
    name = :name
WHERE
    id = :id
    AND user_id = :user_id;

-- name: GetYearNameByID :one
SELECT
    name
FROM
    years
WHERE
    id = :id
    AND user_id = :user_id;