-- name: CreateStudentWithClassCode :exec
INSERT INTO
student_class_codes (student_id, class_code_id, user_id)
VALUES
    (:student_id, :class_code_id, :user_id);