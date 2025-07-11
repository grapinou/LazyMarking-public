-- name: CreateUser :one
INSERT INTO users (username, email, hashpassword)
VALUES (?, ?, ?)
RETURNING *;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = ?;

-- name: GetUserByUsername :one
SELECT * FROM users WHERE username = ?;