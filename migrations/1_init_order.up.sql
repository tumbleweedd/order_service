CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS "order"
(
    uuid         uuid      DEFAULT uuid_generate_v1mc() PRIMARY KEY,
    user_uuid    uuid NOT NULL,
    status       int  NOT NULL,
    payment_type smallint,
    created_at   timestamp default now(),
    updated_at   timestamp default now()
);
CREATE INDEX IF NOT EXISTS idx_order_uuid ON "order" (uuid);

CREATE TABLE IF NOT EXISTS "order_products"
(
    order_product_id bigserial PRIMARY KEY,
    order_uuid       uuid NOT NULL,
    product_uuid     uuid NOT NULL,
    amount           int  NOT NULL,

    CONSTRAINT fk_order_item_id FOREIGN KEY (order_uuid) REFERENCES "order" (uuid)
);