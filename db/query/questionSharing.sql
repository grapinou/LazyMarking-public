-- name: ShareQuestion :execrows
INSERT INTO question_shares(question_id)
SELECT id FROM questions WHERE id = :question_id AND user_id = :user_id
ON CONFLICT(question_id) DO NOTHING;

-- name: UnshareQuestion :execrows
DELETE FROM question_shares WHERE question_id = :question_id
AND EXISTS (SELECT 1 FROM questions WHERE id = :question_id AND user_id = :user_id);

-- name: GetOwnedQuestionSharing :many
SELECT q.id, EXISTS(SELECT 1 FROM question_shares s WHERE s.question_id = q.id) AS shared,
COALESCE(o.source_author, '') AS source_author
FROM questions q LEFT JOIN question_copy_origins o ON o.question_id = q.id
WHERE q.user_id = :user_id;

-- name: GetSharedQuestions :many
SELECT q.id, q.user_id, q.content, q.instruction, u.username AS author,
s.name AS subject_name, y.name AS year_level_name, t.name AS theme_name,
k.name AS skill_name, d.name AS difficulty_name, p.point_value
FROM question_shares sh JOIN questions q ON q.id = sh.question_id
JOIN users u ON u.id = q.user_id
JOIN subjects s ON s.id = q.subject_id AND s.user_id = q.user_id
JOIN year_levels y ON y.id = q.year_level_id AND y.user_id = q.user_id
JOIN themes t ON t.id = q.theme_id AND t.user_id = q.user_id
JOIN skills k ON k.id = q.skill_id AND k.user_id = q.user_id
JOIN difficulties d ON d.id = q.difficulty_id AND d.user_id = q.user_id
JOIN points p ON p.id = q.point_id AND p.user_id = q.user_id
ORDER BY q.id DESC;

-- name: GetSharedQuestion :one
SELECT q.id, q.user_id, q.content, q.instruction, u.username AS author,
s.name AS subject_name, y.name AS year_level_name, t.name AS theme_name,
k.name AS skill_name, d.name AS difficulty_name, p.point_value
FROM question_shares sh JOIN questions q ON q.id = sh.question_id
JOIN users u ON u.id = q.user_id
JOIN subjects s ON s.id = q.subject_id AND s.user_id = q.user_id
JOIN year_levels y ON y.id = q.year_level_id AND y.user_id = q.user_id
JOIN themes t ON t.id = q.theme_id AND t.user_id = q.user_id
JOIN skills k ON k.id = q.skill_id AND k.user_id = q.user_id
JOIN difficulties d ON d.id = q.difficulty_id AND d.user_id = q.user_id
JOIN points p ON p.id = q.point_id AND p.user_id = q.user_id
WHERE q.id = :question_id;

-- name: GetSharedVariants :many
SELECT a.* FROM alt_questions a
JOIN questions q ON q.id = a.question_id AND q.user_id = a.user_id
JOIN question_shares s ON s.question_id = q.id
ORDER BY a.question_id, a.id;
