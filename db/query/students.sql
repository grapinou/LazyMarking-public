-- name: CreateStudent :exec
INSERT INTO
    students (first_name, last_name, user_id)
VALUES
    (:first_name, :last_name, :user_id);


-- name: CreateStudentAndReturnID :one
INSERT INTO
    students (first_name, last_name, user_id)
VALUES
    (:first_name, :last_name, :user_id)
RETURNING id;

-- name: DeleteStudent :execrows
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

-- name: UpdateStudent :execrows
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


-- name: GetStudentsWithClasses :many
SELECT
    s.id           AS student_id,
    s.first_name   AS student_first_name,
    s.last_name    AS student_last_name,
    c.id           AS class_id,
    c.name         AS class_name
FROM students AS s
LEFT JOIN student_class_codes AS sc
    ON s.id = sc.student_id
    AND sc.user_id = s.user_id
LEFT JOIN class_codes AS c
    ON sc.class_code_id = c.id
    AND c.user_id = s.user_id
WHERE s.user_id = :user_id
  AND (:class_filter = '' OR c.name = :class_filter)
ORDER BY c.name ASC, s.last_name ASC, s.first_name ASC;


-- name: DeleteStudentsOnlyInOneClass :execrows
DELETE FROM students
WHERE students.user_id = sqlc.arg(user_id)
AND EXISTS (
    SELECT 1 FROM class_codes AS owned_class
    WHERE owned_class.id = :class_code_id
      AND owned_class.user_id = sqlc.arg(user_id)
)
AND id IN (
    SELECT sc.student_id
    FROM student_class_codes AS sc
    WHERE sc.user_id = sqlc.arg(user_id)
    GROUP BY sc.student_id
    HAVING COUNT(*) = 1
)
AND id IN (
    SELECT sc2.student_id
    FROM student_class_codes AS sc2
    WHERE sc2.class_code_id = :class_code_id
      AND sc2.user_id = sqlc.arg(user_id)
);


-- name: DeleteStudentsWithSeveralClass :execrows
DELETE FROM student_class_codes AS scc
WHERE scc.class_code_id = :class_code_id
  AND scc.user_id = :user_id
  AND EXISTS (
      SELECT 1 FROM class_codes AS owned_class
      WHERE owned_class.id = :class_code_id
        AND owned_class.user_id = sqlc.arg(user_id)
  )
  AND scc.student_id IN (
      SELECT sc.student_id
      FROM student_class_codes AS sc
      JOIN students AS owned_student
        ON owned_student.id = sc.student_id
       AND owned_student.user_id = sqlc.arg(user_id)
      WHERE sc.user_id = sqlc.arg(user_id)
      GROUP BY sc.student_id
      HAVING COUNT(*) > 1
  );
