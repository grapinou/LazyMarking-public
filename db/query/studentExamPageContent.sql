-- name: CreateStudentExamPageContent :exec
INSERT INTO
    student_exam_page_content (student_exam_id, page, content, user_id)
VALUES
    (:student_exam_id, :page, :content, :user_id);