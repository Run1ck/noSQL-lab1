-- name: CreateRequest :one
INSERT INTO requests (user_id) VALUES ($1) RETURNING id, created_at;

-- name: GetRequest :one
SELECT * FROM requests WHERE id = $1;

-- name: GetAllRequests :many
SELECT * FROM requests ORDER BY created_at DESC, id DESC;

-- name: GetRequestsByStatus :many
SELECT * FROM requests WHERE status = $1 ORDER BY created_at DESC, id DESC;

-- name: GetRequestsByUser :many
SELECT * FROM requests WHERE user_id = $1 ORDER BY created_at DESC, id DESC;

-- name: ProcessRequest :execresult
UPDATE requests
SET status = $2, comment = $3, processed_at = $4, processed_by = $5
WHERE id = $1 AND status = 'new';
