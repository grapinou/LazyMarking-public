-- name: CreateStudentWithClassCode :exec
INSERT INTO
student_class_codes (student_id, class_code_id, user_id)
VALUES
    (:student_id, :class_code_id, :user_id);


-- name: GetAllClassCodesByStudentID :many
SELECT
class_code_id
FROM
student_class_codes
WHERE
student_id = :student_id AND user_id = :user_id;


-- name: DeleteStudentClassCodeByStudentID :exec
DELETE FROM
student_class_codes
WHERE
student_id = :student_id AND class_code_id = :class_code_id AND user_id = :user_id;


-- name: ListClassCodesNotAssignedToStudent :many
SELECT
    class_codes.id   AS "class_codes_id",
    class_codes.name AS "class_codes_name"
FROM class_codes AS class_codes
LEFT JOIN student_class_codes AS student_class_codes
    ON class_codes.id = student_class_codes.class_code_id
    AND student_class_codes.student_id = :student_id
WHERE class_codes.user_id = :user_id
  AND student_class_codes.id IS NULL;

-- name: CountStudentsInClass :one
SELECT COUNT(*) AS total
FROM student_class_codes
WHERE class_code_id = :class_code_id
  AND user_id = :user_id;