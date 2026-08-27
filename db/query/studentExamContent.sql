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
    student_exam_content.student_exam_id = :student_exam_id
    AND student_exam_content.user_id = sqlc.arg(user_id)
    AND EXISTS (
        SELECT 1 FROM student_exam
        WHERE student_exam.id = student_exam_content.student_exam_id
          AND student_exam.user_id = sqlc.arg(user_id)
    );
