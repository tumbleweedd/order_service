CREATE TABLE IF NOT EXISTS outbox (
    id SERIAL PRIMARY KEY,
    event_uuid uuid NOT NULL,
    order_uuid uuid NOT NULL,
    send BOOLEAN DEFAULT FALSE
);