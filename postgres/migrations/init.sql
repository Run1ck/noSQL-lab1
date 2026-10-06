CREATE TYPE user_role AS ENUM ('user', 'admin');

CREATE TABLE users (
    id            UUID        PRIMARY KEY DEFAULT uuidv7(),
    login         TEXT        NOT NULL UNIQUE,
    password_hash TEXT        NOT NULL,
    role          user_role   NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TYPE service_kind AS ENUM ('room', 'lab', 'equipment');

CREATE TABLE services (
    id     TEXT         PRIMARY KEY,
    name   TEXT         NOT NULL,
    kind   service_kind NOT NULL,
    active BOOLEAN      NOT NULL DEFAULT TRUE
);

CREATE TYPE request_status AS ENUM ('new', 'approved', 'rejected', 'cancelled');

CREATE TABLE requests (
    id           BIGINT         GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id      UUID           NOT NULL REFERENCES users (id),
    status       request_status NOT NULL DEFAULT 'new',
    comment      TEXT           NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ    NOT NULL DEFAULT now(),
    processed_at TIMESTAMPTZ,
    processed_by UUID           REFERENCES users (id)
);

CREATE INDEX requests_status_idx ON requests (status);
CREATE INDEX requests_user_id_idx ON requests (user_id);

CREATE TABLE request_items (
    request_id BIGINT NOT NULL REFERENCES requests (id) ON DELETE CASCADE,
    service_id TEXT   NOT NULL REFERENCES services (id),
    date       DATE   NOT NULL,
    PRIMARY KEY (request_id, service_id, date)
);

CREATE TABLE bookings (
    service_id TEXT   NOT NULL,
    date       DATE   NOT NULL,
    request_id BIGINT NOT NULL,
    PRIMARY KEY (service_id, date),
    FOREIGN KEY (request_id, service_id, date)
        REFERENCES request_items (request_id, service_id, date) ON DELETE CASCADE
);

CREATE INDEX bookings_date_idx ON bookings (date);
