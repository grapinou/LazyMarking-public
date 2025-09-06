-- name: GetExamsAllInfos :many
SELECT
    exams.id,
    exams.name AS exam_name,
    years.name AS year_name,
    periods.name AS period_name,
    qcm.name AS qcm_name,
    class_codes.name AS class_code_name
FROM
    exams
    JOIN years ON years.id = exams.year_id
    JOIN periods ON periods.id = exams.period_id
    JOIN qcm ON qcm.id = exams.qcm_id
    JOIN class_codes ON class_codes.id = exams.class_code_id
WHERE
    exams.user_id = :user_id
ORDER BY
    exams.id DESC;

-- name: CreateExam :exec
INSERT INTO
    exams (
        name,
        qcm_id,
        class_code_id,
        period_id,
        year_id,
        user_id
    )
VALUES
    (
        :name,
        :qcm_id,
        :class_code_id,
        :period_id,
        :year_id,
        :user_id
    );

-- name: GetExamByID :one
SELECT
    *
FROM
    exams
WHERE
    id = :id
    and user_id = :user_id;

-- name: UpdateExam :exec
UPDATE
    exams
SET
    name = :name,
    qcm_id = :qcm_id,
    class_code_id = :class_code_id,
    period_id = :period_id,
    year_id = :year_id
WHERE
    id = :id
    AND user_id = :user_id;

-- name: DeleteExam :exec
DELETE FROM
    exams
WHERE
    id = :id
    AND user_id = :user_id;

-- name: GetExamNameAndClassCodeName :one
SELECT
    exams.name AS exam_name,
    class_codes.name AS class_name
FROM
    exams
    JOIN class_codes ON exams.class_code_id = class_codes.id
WHERE
    exams.id = :id
    AND exams.user_id = :user_id;