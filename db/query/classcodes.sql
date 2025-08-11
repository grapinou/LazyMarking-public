-- name: CreateClassCode :exec
INSERT INTO
    class_codes (name, user_id)
VALUES
    (:name, :user_id);

-- name: DeleteClassCode :exec
DELETE FROM
    class_codes
WHERE
    id = :id
    AND user_id = :user_id;

-- name: GetAllClassCodes :many
SELECT
    *
FROM
    class_codes
WHERE
    user_id = :user_id
ORDER BY
    name;

-- name: UpdateClassCode :exec
UPDATE
    class_codes
SET
    name = :name
WHERE
    id = :id
    AND user_id = :user_id;

-- name: GetClassCodeNameByID :one
SELECT
    name
FROM
    class_codes
WHERE
    id = :id
    AND user_id = :user_id;


-- name: ListClassCodesByUser :many
SELECT
    id,
    name
FROM class_codes
WHERE user_id = :user_id
ORDER BY name;