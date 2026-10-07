-- name: AddItemsToRequest :copyfrom
INSERT INTO request_items (request_id, service_id, date) VALUES ($1, $2, $3);

-- name: GetRequestItems :many
SELECT * FROM request_items
WHERE request_id = ANY(@request_ids::bigint[])
ORDER BY request_id, date, service_id;

-- name: GetRequestInfo :many
SELECT services.name, services.kind, date FROM request_items INNER JOIN services ON services.id = service_id WHERE request_id = $1;
