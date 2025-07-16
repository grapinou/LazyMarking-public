-- name: CreateYearLevel :exec
INSERT INTO
    year_levels (name, user_id)
VALUES
    (:name, :user_id);

-- name: DeleteYearLevel :exec
DELETE FROM
    year_levels
WHERE
    id = :id
    AND user_id = :user_id;

-- name: GetAllYearLevels :many
SELECT
    *
FROM
    year_levels
WHERE
    user_id = :user_id
ORDER BY
    name;

-- name: UpdateYearLevel :exec
UPDATE
    year_levels
SET
    name = :name
WHERE
    id = :id
    AND user_id = :user_id;