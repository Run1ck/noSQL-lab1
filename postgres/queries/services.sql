-- name: GetAllServices :many
SELECT * FROM services ORDER BY id;

-- name: GetService :one
SELECT * FROM services WHERE id = $1;

-- name: GetServicesByKind :many
SELECT * FROM services WHERE kind = $1 ORDER BY id;

-- name: CreateService :execresult
INSERT INTO services (id, name, kind, active) VALUES ($1, $2, $3, $4);

-- name: UpsertService :exec
INSERT INTO services (id, name, kind, active) VALUES ($1, $2, $3, $4)
ON CONFLICT (id) DO UPDATE
SET name = EXCLUDED.name, kind = EXCLUDED.kind, active = EXCLUDED.active;

-- name: SearchForService :many
SELECT * FROM services WHERE name ILIKE '%' || @query::text || '%' ORDER BY id;

-- name: SetServiceStatus :execresult
UPDATE services SET active = $2 WHERE id = $1;
