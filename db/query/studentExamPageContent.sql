-- name: CreateStudentExamPageContent :exec
INSERT INTO
    student_exam_page_content (student_exam_id, page, content, user_id)
VALUES
    (:student_exam_id, :page, :content, :user_id);

-- name: SetStudentExamPageReference :execrows
UPDATE student_exam_page_content
SET
    reference_storage_key = sqlc.arg(reference_storage_key),
    reference_width = sqlc.arg(reference_width),
    reference_height = sqlc.arg(reference_height),
    reference_dpi = sqlc.arg(reference_dpi),
    reference_sha256 = sqlc.arg(reference_sha256)
WHERE student_exam_page_content.student_exam_id = sqlc.arg(student_exam_id)
  AND student_exam_page_content.page = sqlc.arg(page)
  AND student_exam_page_content.user_id = sqlc.arg(user_id)
  AND sqlc.arg(reference_storage_key) = printf(
      'references/student-exam-%d/page-%d.png',
      sqlc.arg(student_exam_id),
      sqlc.arg(page)
  )
  AND (SELECT COUNT(*)
       FROM student_exam_page_content AS page_rows
       WHERE page_rows.student_exam_id = sqlc.arg(student_exam_id)
         AND page_rows.page = sqlc.arg(page)
         AND page_rows.user_id = sqlc.arg(user_id)) = 1
  AND EXISTS (
      SELECT 1
      FROM student_exam AS se
      JOIN exams_generated AS eg
        ON eg.id = se.exam_generated_id
       AND eg.user_id = se.user_id
      WHERE se.id = student_exam_page_content.student_exam_id
        AND se.user_id = sqlc.arg(user_id)
  )
  AND (
      student_exam_page_content.reference_storage_key IS NULL
      OR (
          student_exam_page_content.reference_storage_key = sqlc.arg(reference_storage_key)
          AND student_exam_page_content.reference_width = sqlc.arg(reference_width)
          AND student_exam_page_content.reference_height = sqlc.arg(reference_height)
          AND student_exam_page_content.reference_dpi = sqlc.arg(reference_dpi)
          AND student_exam_page_content.reference_sha256 = sqlc.arg(reference_sha256)
      )
  );

-- name: GetStudentExamPageReference :one
SELECT
    eg.id AS generation_id,
    u.username,
    sep.student_exam_id,
    sep.page,
    sep.reference_storage_key,
    sep.reference_width,
    sep.reference_height,
    sep.reference_dpi,
    sep.reference_sha256
FROM student_exam_page_content AS sep
JOIN student_exam AS se
  ON se.id = sep.student_exam_id
 AND se.user_id = sep.user_id
JOIN exams_generated AS eg
  ON eg.id = se.exam_generated_id
 AND eg.user_id = se.user_id
JOIN users AS u
  ON u.id = eg.user_id
WHERE sep.student_exam_id = sqlc.arg(student_exam_id)
  AND sep.page = sqlc.arg(page)
  AND sep.user_id = sqlc.arg(user_id)
  AND (SELECT COUNT(*)
       FROM student_exam_page_content AS page_rows
       WHERE page_rows.student_exam_id = sqlc.arg(student_exam_id)
         AND page_rows.page = sqlc.arg(page)
         AND page_rows.user_id = sqlc.arg(user_id)) = 1;

-- name: ListExamGenerationPageReferences :many
SELECT
    sep.student_exam_id,
    sep.page
FROM exams_generated AS eg
JOIN student_exam AS se
  ON se.exam_generated_id = eg.id
 AND se.user_id = eg.user_id
JOIN student_exam_page_content AS sep
  ON sep.student_exam_id = se.id
 AND sep.user_id = se.user_id
WHERE eg.id = sqlc.arg(generation_id)
  AND eg.user_id = sqlc.arg(user_id)
ORDER BY sep.student_exam_id, sep.page;

-- name: GetExamGenerationReferenceCoverage :one
SELECT
    COUNT(*) AS expected_pages,
    COALESCE(SUM(CASE WHEN
        sep.reference_storage_key IS NOT NULL
        AND sep.reference_width IS NOT NULL
        AND sep.reference_height IS NOT NULL
        AND sep.reference_dpi IS NOT NULL
        AND sep.reference_sha256 IS NOT NULL
        THEN 1 ELSE 0 END), 0) AS referenced_pages,
    COALESCE((
        SELECT COUNT(*)
        FROM (
            SELECT duplicate.student_exam_id, duplicate.page
            FROM student_exam_page_content AS duplicate
            JOIN student_exam AS duplicate_exam
              ON duplicate_exam.id = duplicate.student_exam_id
             AND duplicate_exam.user_id = duplicate.user_id
            WHERE duplicate_exam.exam_generated_id = eg.id
              AND duplicate.user_id = eg.user_id
            GROUP BY duplicate.student_exam_id, duplicate.page, duplicate.user_id
            HAVING COUNT(*) != 1
        )
    ), 0) AS ambiguous_pages
FROM exams_generated AS eg
JOIN student_exam AS se
  ON se.exam_generated_id = eg.id
 AND se.user_id = eg.user_id
JOIN student_exam_page_content AS sep
  ON sep.student_exam_id = se.id
 AND sep.user_id = se.user_id
WHERE eg.id = sqlc.arg(generation_id)
  AND eg.user_id = sqlc.arg(user_id)
GROUP BY eg.id;

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
