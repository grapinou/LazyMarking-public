-- name: CreateStudentExam :one
INSERT INTO
    student_exam (exam_generated_id, student_id, user_id)
VALUES
    (:exam_generated_id, :student_id, :user_id) RETURNING id;