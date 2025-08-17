-- name: GetExamsAllInfos :many
SELECT 
exams.id,
exams.name AS exam_name,
years.value AS year_value,
periods.name AS period_name
FROM exams
JOIN years ON years.id = exams.year_id
JOIN periods ON periods.id = exams.period_id
JOIN qcm ON qcm.id = exams.qcm_id
JOIN class_codes ON class_codes.id = exams.class_code_id
WHERE exams.user_id = :user_id
ORDER BY exams.id DESC;




  