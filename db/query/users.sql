-- name: CreateUser :exec
INSERT INTO users (username, email, hashpassword)
VALUES (:username, :email, :hashpassword);

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = :email;

-- name: GetUserByUsername :one
SELECT * FROM users WHERE username = :username;

-- name: UpdateUserPassword :execrows
UPDATE users 
SET hashpassword = :hashpassword 
WHERE id = :id;
