-- name: CreateStudentWithClassCode :execrows
INSERT INTO
student_class_codes (student_id, class_code_id, user_id)
SELECT :student_id, :class_code_id, :user_id
WHERE EXISTS (
    SELECT 1 FROM students
    WHERE id = :student_id AND user_id = sqlc.arg(user_id)
)
AND EXISTS (
    SELECT 1 FROM class_codes
    WHERE id = :class_code_id AND user_id = sqlc.arg(user_id)
);


-- name: GetAllClassCodesByStudentID :many
SELECT
class_code_id
FROM
student_class_codes
WHERE
student_id = :student_id AND user_id = :user_id;


-- name: ListStudentClassCodesWithNames :many
SELECT
    class_codes.id AS class_code_id,
    class_codes.name AS class_code_name
FROM students AS students
JOIN student_class_codes AS student_class_codes
    ON student_class_codes.student_id = students.id
    AND student_class_codes.user_id = students.user_id
JOIN class_codes AS class_codes
    ON class_codes.id = student_class_codes.class_code_id
    AND class_codes.user_id = students.user_id
WHERE students.id = :student_id
  AND students.user_id = :user_id
ORDER BY student_class_codes.id ASC;


-- name: DeleteStudentClassCodeByStudentID :execrows
DELETE FROM
student_class_codes
WHERE
student_class_codes.student_id = sqlc.arg(student_id)
AND student_class_codes.class_code_id = sqlc.arg(class_code_id)
AND student_class_codes.user_id = sqlc.arg(user_id)
AND EXISTS (
    SELECT 1 FROM students
    WHERE id = sqlc.arg(student_id) AND user_id = sqlc.arg(user_id)
)
AND EXISTS (
    SELECT 1 FROM class_codes
    WHERE id = sqlc.arg(class_code_id) AND user_id = sqlc.arg(user_id)
)
AND EXISTS (
    SELECT 1
    FROM student_class_codes AS remaining_relation
    WHERE remaining_relation.student_id = sqlc.arg(student_id)
      AND remaining_relation.user_id = sqlc.arg(user_id)
      AND remaining_relation.class_code_id <> sqlc.arg(class_code_id)
);


-- name: ListClassCodesNotAssignedToStudent :many
SELECT
    class_codes.id   AS "class_codes_id",
    class_codes.name AS "class_codes_name"
FROM class_codes AS class_codes
LEFT JOIN student_class_codes AS student_class_codes
    ON class_codes.id = student_class_codes.class_code_id
    AND student_class_codes.student_id = :student_id
WHERE class_codes.user_id = :user_id
  AND EXISTS (
      SELECT 1 FROM students
      WHERE id = :student_id AND user_id = sqlc.arg(user_id)
  )
  AND student_class_codes.id IS NULL;

-- name: CountStudentsInClass :one
SELECT COUNT(*) AS total
FROM student_class_codes
WHERE class_code_id = :class_code_id
  AND user_id = :user_id;


-- name: GetAllStudentsByClassCodeID :many
SELECT students.*
FROM students
JOIN student_class_codes 
    ON students.id = student_class_codes.student_id
WHERE student_class_codes.class_code_id = :class_code_id
  AND students.user_id = :user_id
  AND student_class_codes.user_id = :user_id;
