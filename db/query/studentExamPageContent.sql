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
    student_exam_page_content.student_exam_id = :student_exam_id
    AND student_exam_page_content.page = :page
    AND student_exam_page_content.user_id = sqlc.arg(user_id)
    AND EXISTS (
        SELECT 1 FROM student_exam
        WHERE student_exam.id = student_exam_page_content.student_exam_id
          AND student_exam.user_id = sqlc.arg(user_id)
    );
