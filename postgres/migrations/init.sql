CREATE TABLE users (
    login         TEXT        PRIMARY KEY,
    name          TEXT        NOT NULL,
    password_hash TEXT        NOT NULL,
    role          TEXT        NOT NULL CHECK (role IN ('user', 'admin')),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE services (
    id     TEXT  PRIMARY KEY ,
    name   TEXT    NOT NULL,
    kind   TEXT    NOT NULL CHECK (kind IN ('room', 'lab', 'equipment')),
    active BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE TABLE requests (
    id           BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_login   TEXT        NOT NULL REFERENCES users (login),
    status       TEXT        NOT NULL DEFAULT 'new'
                 CHECK (status IN ('new', 'approved', 'rejected', 'cancelled')),
    comment      TEXT        NOT NULL DEFAULT '',
    response_comment      TEXT        NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    processed_at TIMESTAMPTZ,
    processed_by TEXT REFERENCES users (login)
);

CREATE INDEX requests_status_idx ON requests (status);

CREATE TABLE request_items (
    request_id BIGINT NOT NULL REFERENCES requests (id) ON DELETE CASCADE,
    service_id TEXT   NOT NULL REFERENCES services (id),
    date       DATE   NOT NULL,
    PRIMARY KEY (request_id, service_id, date)
);

-- Подтверждённые брони: услуга занята на весь день. День занимает только
-- одобренная заявка — корзина и заявки в статусе new ничего не блокируют,
-- поэтому на один день может быть несколько заявок. Строки вставляются
-- в одной транзакции с переводом заявки в approved; первичный ключ не даст
-- одобрить вторую заявку на тот же день.
CREATE TABLE bookings (
    service_id TEXT   NOT NULL,
    date       DATE   NOT NULL,
    request_id BIGINT NOT NULL,
    PRIMARY KEY (service_id, date),
    FOREIGN KEY (request_id, service_id, date)
        REFERENCES request_items (request_id, service_id, date) ON DELETE CASCADE
);
