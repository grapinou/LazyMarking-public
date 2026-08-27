-- name: CreatePeriod :exec
INSERT INTO
    periods (name, user_id)
VALUES
    (:name, :user_id);

-- name: DeletePeriod :execrows
DELETE FROM
    periods
WHERE
    id = :id
    AND user_id = :user_id;

-- name: GetAllPeriods :many
SELECT
    *
FROM
    periods
WHERE
    user_id = :user_id
ORDER BY
    name;

-- name: UpdatePeriod :execrows
UPDATE
    periods
SET
    name = :name
WHERE
    id = :id
    AND user_id = :user_id;

-- name: GetPeriodNameByID :one
SELECT
    name
FROM
    periods
WHERE
    id = :id
    AND user_id = :user_id;
