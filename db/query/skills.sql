-- name: CreateSkill :exec
INSERT INTO
    skills (name, user_id)
VALUES
    (:name, :user_id);

-- name: DeleteSkill :exec
DELETE FROM
    skills
WHERE
    id = :id
    AND user_id = :user_id;

-- name: GetAllSkills :many
SELECT
    *
FROM
    skills
WHERE
    user_id = :user_id
ORDER BY
    name;

-- name: UpdateSkill :exec
UPDATE
    skills
SET
    name = :name
WHERE
    id = :id
    AND user_id = :user_id;