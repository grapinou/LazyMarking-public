-- name: CreateDifficulty :exec
INSERT INTO
    difficulties (name, user_id)
VALUES
    (:name, :user_id);

-- name: DeleteDifficulty :exec
DELETE FROM
    difficulties
WHERE
    id = :id
    AND user_id = :user_id;

-- name: GetAllDifficulties :many
SELECT
    *
FROM
    difficulties
WHERE
    user_id = :user_id
ORDER BY
    name;

-- name: UpdateDifficulty :exec
UPDATE
    difficulties
SET
    name = :name
WHERE
    id = :id
    AND user_id = :user_id;