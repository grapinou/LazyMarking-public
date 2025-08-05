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

-- name: UpdateQuestion :exec
UPDATE
    questions
SET
    subject_id = :subject_id,
    theme_id = :theme_id,
    year_level_id = :year_level_id,
    skill_id = :skill_id,
    difficulty_id = :difficulty_id,
    point_id = :point_id,
    content = :content
WHERE
    id = :id
    AND user_id = :user_id;

-- name: DeleteQuestion :exec
DELETE FROM
    questions
WHERE
    id = :id
    AND user_id = :user_id;

-- name: GetAltQuestionIDsWithImage :many
SELECT
    alt_questions.id
FROM
    alt_questions
    JOIN alt_images ON alt_questions.id = alt_images.alt_question_id
WHERE
    alt_questions.question_id = ?;