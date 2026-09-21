-- name: ShareQCM :execrows
INSERT INTO qcm_shares(qcm_id)
SELECT id FROM qcm WHERE id = :qcm_id AND user_id = :user_id
ON CONFLICT(qcm_id) DO NOTHING;

-- name: UnshareQCM :execrows
DELETE FROM qcm_shares WHERE qcm_id = :qcm_id
AND EXISTS (SELECT 1 FROM qcm WHERE id = :qcm_id AND user_id = :user_id);

-- name: GetOwnedQCMSharing :many
SELECT q.id, EXISTS(SELECT 1 FROM qcm_shares s WHERE s.qcm_id = q.id) AS shared,
COALESCE(o.source_author, '') AS source_author
FROM qcm q LEFT JOIN qcm_copy_origins o ON o.qcm_id = q.id
WHERE q.user_id = :user_id;

-- name: GetSharedQCMs :many
SELECT q.id, q.name, q.user_id, u.username AS author,
(SELECT COUNT(*) FROM qcm_questions c WHERE c.qcm_id = q.id AND c.user_id = q.user_id) AS question_count
FROM qcm_shares sh JOIN qcm q ON q.id = sh.qcm_id
JOIN users u ON u.id = q.user_id
ORDER BY q.id DESC;

-- name: GetSharedQCM :one
SELECT q.id, q.name, q.user_id, u.username AS author,
(SELECT COUNT(*) FROM qcm_questions c WHERE c.qcm_id = q.id AND c.user_id = q.user_id) AS question_count
FROM qcm_shares sh JOIN qcm q ON q.id = sh.qcm_id
JOIN users u ON u.id = q.user_id
WHERE q.id = :qcm_id;

-- name: GetSharedQCMClassifications :many
SELECT DISTINCT c.id AS qcm_id, s.name AS subject_name, y.name AS year_level_name
FROM qcm_shares sh JOIN qcm c ON c.id = sh.qcm_id
JOIN qcm_questions rel ON rel.qcm_id = c.id AND rel.user_id = c.user_id
JOIN questions q ON q.id = rel.question_id AND q.user_id = c.user_id
JOIN subjects s ON s.id = q.subject_id AND s.user_id = q.user_id
JOIN year_levels y ON y.id = q.year_level_id AND y.user_id = q.user_id
ORDER BY c.id, s.name, y.name;

-- name: GetSharedQCMComposition :many
SELECT rel.question_id, rel.position
FROM qcm_shares sh JOIN qcm c ON c.id = sh.qcm_id
JOIN qcm_questions rel ON rel.qcm_id = c.id AND rel.user_id = c.user_id
JOIN questions q ON q.id = rel.question_id AND q.user_id = c.user_id
WHERE c.id = :qcm_id
ORDER BY rel.position;

-- name: GetSharedQCMFamily :one
SELECT q.id, q.user_id, q.content, q.instruction, u.username AS author,
s.name AS subject_name, y.name AS year_level_name, t.name AS theme_name,
k.name AS skill_name, d.name AS difficulty_name, p.point_value
FROM questions q
JOIN users u ON u.id = q.user_id
JOIN subjects s ON s.id = q.subject_id AND s.user_id = q.user_id
JOIN year_levels y ON y.id = q.year_level_id AND y.user_id = q.user_id
JOIN themes t ON t.id = q.theme_id AND t.user_id = q.user_id
JOIN skills k ON k.id = q.skill_id AND k.user_id = q.user_id
JOIN difficulties d ON d.id = q.difficulty_id AND d.user_id = q.user_id
JOIN points p ON p.id = q.point_id AND p.user_id = q.user_id
WHERE q.id = :question_id AND EXISTS (
    SELECT 1 FROM qcm_shares sh JOIN qcm c ON c.id = sh.qcm_id
    JOIN qcm_questions rel ON rel.qcm_id = c.id AND rel.user_id = c.user_id
    WHERE c.id = :qcm_id AND c.user_id = q.user_id AND rel.question_id = q.id
);
