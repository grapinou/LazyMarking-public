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





-- name: GetFilteredQuestions :many
SELECT 
    q.id,
    q.content,
    s.name  AS subject_name,
    t.name  AS theme_name,
    y.name  AS year_level_name,
    sk.name AS skill_name,
    d.name  AS difficulty_name,
    p.point_value AS point_value
FROM (
    SELECT *
    FROM questions
    WHERE questions.user_id = :user_id
      AND (CAST(sqlc.narg('subject_id') AS INTEGER) IS NULL OR subject_id     = CAST(sqlc.narg('subject_id') AS INTEGER))
      AND (CAST(sqlc.narg('theme_id') AS INTEGER) IS NULL OR theme_id         = CAST(sqlc.narg('theme_id') AS INTEGER))
      AND (CAST(sqlc.narg('year_level_id') AS INTEGER) IS NULL OR year_level_id = CAST(sqlc.narg('year_level_id') AS INTEGER))
      AND (CAST(sqlc.narg('skill_id') AS INTEGER) IS NULL OR skill_id         = CAST(sqlc.narg('skill_id') AS INTEGER))
      AND (CAST(sqlc.narg('difficulty_id') AS INTEGER) IS NULL OR difficulty_id = CAST(sqlc.narg('difficulty_id') AS INTEGER))
      AND (CAST(sqlc.narg('point_id') AS INTEGER) IS NULL OR point_id         = CAST(sqlc.narg('point_id') AS INTEGER))
) q
JOIN subjects     s  ON q.subject_id     = s.id
JOIN themes       t  ON q.theme_id       = t.id
JOIN year_levels  y  ON q.year_level_id  = y.id
JOIN skills       sk ON q.skill_id       = sk.id
JOIN difficulties d  ON q.difficulty_id  = d.id
JOIN points       p  ON q.point_id       = p.id
ORDER BY q.id;

-- name: GetTagsByQuestionID :one
SELECT
    s.id AS subject_id,
    s.name AS subject_name,

    t.id AS theme_id,
    t.name AS theme_name,

    y.id AS year_level_id,
    y.name AS year_level_name,

    sk.id AS skill_id,
    sk.name AS skill_name,

    d.id AS difficulty_id,
    d.name AS difficulty_name,

    p.id AS point_id,
    p.point_value AS point_value
FROM questions q
JOIN subjects s ON q.subject_id = s.id
JOIN themes t ON q.theme_id = t.id
JOIN year_levels y ON q.year_level_id = y.id
JOIN skills sk ON q.skill_id = sk.id
JOIN difficulties d ON q.difficulty_id = d.id
JOIN points p ON q.point_id = p.id
WHERE q.id = :question_id AND q.user_id = :user_id;


-- name: GetRandomQuestionByQuestionID :one
WITH pool AS (
  SELECT
    q.id      AS item_id,
    q.content AS content,
    0         AS is_alt
  FROM questions q
  WHERE q.id = :question_id

  UNION ALL

  SELECT
    a.id      AS item_id,
    a.content AS content,
    1         AS is_alt
  FROM alt_questions a
  WHERE a.question_id = :question_id
)
SELECT item_id, content, is_alt
FROM pool
ORDER BY RANDOM()
LIMIT 1;
