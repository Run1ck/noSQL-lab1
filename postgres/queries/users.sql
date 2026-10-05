-- name: CheckPassword :one
SELECT 1 FROM users WHERE login = $1 AND password_hash = $2;

-- name: CreateUser :execresult
INSERT INTO users (login, name, password_hash, role) VALUES ($1, $2, $3, $4) ON CONFLICT DO NOTHING;

-- name: GetInfoByLogin :one
SELECT name,role,created_at FROM users WHERE login = $1;
