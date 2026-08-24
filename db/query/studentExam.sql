-- name: CreateStudentExam :one
INSERT INTO
    student_exam (exam_generated_id, student_id, user_id)
VALUES
    (:exam_generated_id, :student_id, :user_id) RETURNING id;

-- name: GetStudentNameByStudentExamID :one
SELECT
    students.first_name,
    students.last_name
FROM
    students
    JOIN student_exam ON student_exam.student_id = students.id
WHERE
    student_exam.id = :id
    AND student_exam.user_id = :user_id
    AND students.user_id = :user_id;
