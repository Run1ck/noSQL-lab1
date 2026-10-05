-- name: GetAllServices :many
SELECT * FROM services;

-- name: GetServicesByKind :many
SELECT * FROM services WHERE kind = $1;

-- name: CreateService :execresult
INSERT INTO services (id, name, kind, active) VALUES ($1, $2, $3, $4);

-- name: SearchForService :many
SELECT * FROM services WHERE name LIKE $1;

-- name: SetServiceStatus :execresult
UPDATE services SET active = $2 WHERE id = $1;
