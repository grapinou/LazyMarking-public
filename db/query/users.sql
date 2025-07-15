-- name: CreateUser :exec
INSERT INTO users (username, email, hashpassword)
VALUES (?, ?, ?);

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = ?;

-- name: GetUserByUsername :one
SELECT * FROM users WHERE username = ?;

-- name: UpdateUserPassword :exec
UPDATE users 
SET hashpassword = :hashpassword 
WHERE id = :id;