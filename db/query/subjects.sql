-- name: CreateSubject :exec
INSERT INTO
    subjects (name, user_id)
VALUES
    (:name, :user_id);

-- name: DeleteSubject :exec
DELETE FROM
    subjects
WHERE
    id = :id
    AND user_id = :user_id;

-- name: GetAllSubjects :many
SELECT
    *
FROM
    subjects
WHERE
    user_id = :user_id
ORDER BY
    name;

-- name: UpdateSubject :exec
UPDATE
    subjects
SET
    name = :name
WHERE
    id = :id
    AND user_id = :user_id;

-- name: GetSubjectNameByID :one
SELECT
    name
FROM
    subjects
WHERE
    id = :id
    AND user_id = :user_id;