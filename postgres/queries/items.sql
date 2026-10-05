-- name: AddItemsToRequest :copyfrom
INSERT INTO request_items (request_id, service_id, date) VALUES ($1, $2, $3);

-- name: GetRequestInfo :many
SELECT services.name, services.kind, date FROM request_items INNER JOIN services ON services.id = service_id WHERE request_id = $1;
