-- name: CreateStudentExamContent :exec
INSERT INTO
    student_exam_content (student_exam_id, page_tot, content, user_id)
VALUES
    (:student_exam_id, :page_tot, :content, :user_id);

-- name: GetStudentContentExam :one
SELECT
    page_tot,
    content
FROM
    student_exam_content
WHERE
    student_exam_id = :student_exam_id
    AND user_id = :user_id;