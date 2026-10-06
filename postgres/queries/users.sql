-- name: CreateUser :execresult
-- 0 затронутых строк — логин занят (ErrLoginTaken).
INSERT INTO users (id, login, password_hash, role, created_at)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (login) DO NOTHING;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByLogin :one
SELECT * FROM users WHERE login = $1;
