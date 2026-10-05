-- name: GetAllRequests :many
SELECT * FROM requests;

-- name: GetRequestsByStatus :many
SELECT * FROM requests WHERE status = $1;

-- name: CreateRequest :one
INSERT INTO requests (user_login) VALUES ($1) RETURNING id;

-- name: CancelRequest :execresult
UPDATE requests SET status = 'cancelled' WHERE id = $1 AND user_login = $2;

-- name: ProcessRequest :execresult
UPDATE requests SET status = $2, response_comment = $3, processed_at = NOW(), processed_by = $4 WHERE id = $1;
