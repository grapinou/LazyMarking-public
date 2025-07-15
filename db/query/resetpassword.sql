-- name: CreateResetPassword :exec
INSERT INTO password_resets (
    user_id, token, expires_at
) VALUES (
    ?, ?, ?
);

-- name: GetResetPasswordByToken :one
SELECT * FROM password_resets
WHERE token = ? AND used = FALSE AND expires_at > CURRENT_TIMESTAMP;

-- name: MarkResetPasswordTokenUsed :exec
UPDATE password_resets
SET used = TRUE
WHERE token = ?;

-- name: DeleteExpiredResetTokens :exec
DELETE FROM password_resets
WHERE expires_at <= CURRENT_TIMESTAMP OR used = TRUE;