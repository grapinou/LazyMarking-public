-- name: CreateResetPassword :exec
INSERT INTO password_resets (
    user_id, token, expires_at
) VALUES (
    :user_id, :token, :expires_at
);

-- name: GetResetPasswordByToken :one
SELECT * FROM password_resets
WHERE token = :token AND used = FALSE AND unixepoch(expires_at) > unixepoch('now');

-- name: MarkResetPasswordTokenUsed :exec
UPDATE password_resets
SET used = TRUE
WHERE token = :token;

-- name: MarkAllResetPasswordTokensUsedForUser :exec
UPDATE password_resets
SET used = TRUE
WHERE user_id = :user_id;

-- name: DeleteExpiredResetTokens :exec
DELETE FROM password_resets
WHERE unixepoch(expires_at) <= unixepoch('now') OR used = TRUE;
