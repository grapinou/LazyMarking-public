-- name: GetAllQuestions :many
SELECT
    *
FROM
    questions
WHERE
    user_id = :user_id
ORDER BY
    id DESC;

-- name: CreateQuestion :exec
INSERT INTO
    questions (
        subject_id,
        theme_id,
        year_level_id,
        skill_id,
        difficulty_id,
        point_id,
        content,
        user_id
    )
VALUES
    (
        :subject_id,
        :theme_id,
        :year_level_id,
        :skill_id,
        :difficulty_id,
        :point_id,
        :content,
        :user_id
    );

-- name: GetQuestionByID :one
SELECT
    *
FROM
    questions
WHERE
    id = :id
    AND user_id = :user_id;