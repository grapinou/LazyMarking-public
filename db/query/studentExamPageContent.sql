-- name: CreateStudentExamPageContent :exec
INSERT INTO
    student_exam_page_content (student_exam_id, page, content, user_id)
VALUES
    (:student_exam_id, :page, :content, :user_id);

-- name: GetPageContent :one
SELECT
    content
FROM
    student_exam_page_content
WHERE
    student_exam_id = :student_exam_id
    AND page = :page
    AND user_id = :user_id;