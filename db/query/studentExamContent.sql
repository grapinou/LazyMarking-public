-- name: CreateStudentExamContent :exec
INSERT INTO
    student_exam_content (student_exam_id, page_tot, content, user_id)
VALUES
    (:student_exam_id, :page_tot, :content, :user_id);