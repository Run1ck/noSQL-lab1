-- name: CheckServiceAvailability :one
SELECT 1 FROM bookings WHERE service_id = $1 AND date = $2;

-- name: ApproveBooking :execresult
INSERT INTO bookings (service_id, date, request_id) VALUES ($2, $3, $1) ON CONFLICT DO NOTHING;

-- name: CancelBooking :execresult
DELETE FROM bookings WHERE request_id = $1;

-- CREATE TABLE bookings (
--     service_id TEXT   NOT NULL,
--     date       DATE   NOT NULL,
--     request_id BIGINT NOT NULL,
--     PRIMARY KEY (service_id, date),
--     FOREIGN KEY (request_id, service_id, date)
--         REFERENCES request_items (request_id, service_id, date) ON DELETE CASCADE
-- );
