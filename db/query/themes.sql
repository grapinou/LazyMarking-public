-- name: CreateTheme :exec
INSERT INTO
    themes (name, user_id)
VALUES
    (:name, :user_id);

-- name: DeleteTheme :exec
DELETE FROM
    themes
WHERE
    id = :id
    AND user_id = :user_id;

-- name: GetAllThemes :many
SELECT
    *
FROM
    themes
WHERE
    user_id = :user_id
ORDER BY
    name;

-- name: UpdateTheme :exec
UPDATE
    themes
SET
    name = :name
WHERE
    id = :id
    AND user_id = :user_id;

-- name: GetThemeNameByID :one
SELECT
    name
FROM
    themes
WHERE
    id = :id
    AND user_id = :user_id;