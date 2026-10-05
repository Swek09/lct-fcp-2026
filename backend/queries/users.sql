-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;
-- name: GetUser :one
SELECT * FROM users WHERE id = $1;
-- name: CreateUser :exec
INSERT INTO users(id,email,password_hash,role,name) VALUES($1,$2,$3,$4,$5);
-- name: VerifyUser :exec
UPDATE users SET verified = true WHERE id = $1;
