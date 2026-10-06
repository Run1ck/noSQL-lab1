-- name: IsServiceBooked :one
SELECT EXISTS (SELECT 1 FROM bookings WHERE service_id = $1 AND date = $2);

-- name: GetBookingsByDate :many
SELECT * FROM bookings WHERE date = $1 ORDER BY service_id;

-- name: CreateBookingsForRequest :exec
INSERT INTO bookings (service_id, date, request_id)
SELECT service_id, date, request_id FROM request_items WHERE request_items.request_id = $1;
