-- name: CreateStudent :exec
INSERT INTO
    students (first_name, last_name, user_id)
VALUES
    (:first_name, :last_name, :user_id);

-- name: DeleteStudent :exec
DELETE FROM
    students
WHERE
    id = :id
    AND user_id = :user_id;

-- name: GetAllStudents :many
SELECT
    *
FROM
    students
WHERE
    user_id = :user_id;

-- name: UpdateStudent :exec
UPDATE
    students
SET
    first_name = :first_name,
    last_name = :last_name
WHERE
    id = :id
    AND user_id = :user_id;

-- name: GetStudentByID :one
SELECT
*
FROM
    students
WHERE
    id = :id
    AND user_id = :user_id;

-- name: GetStudentIDByNameAndUserID :one
SELECT
id
FROM
students
WHERE
first_name = :first_name AND last_name = :last_name AND user_id = :user_id;


-- name: GetAllStudentsWithClassCodesNames :many
SELECT
    students.id AS student_id,
    students.first_name,
    students.last_name,
    class_codes.id AS class_code_id,
    class_codes.name AS class_code_name
FROM
    students
JOIN
    student_class_codes ON students.id = student_class_codes.student_id
JOIN
    class_codes ON student_class_codes.class_code_id = class_codes.id
WHERE
    students.user_id = :user_id;