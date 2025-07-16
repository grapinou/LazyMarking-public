-- name: GetAllQuestions :many
SELECT
    *
FROM
    questions
WHERE
    user_id = :user_id
ORDER BY
    id DESC;