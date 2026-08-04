CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TYPE order_status AS ENUM (
    'PENDING',
    'CONFIRMED',
    'PAYMENT_PROCESSING',
    'PAID',
    'PAYMENT_FAILED',
    'CANCELLED',
    'SHIPPED',
    'DELIVERED'
);

CREATE TABLE orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID NOT NULL,
    idempotency_key VARCHAR(128) NOT NULL,
    request_hash CHAR(64) NOT NULL,
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    unit_price BIGINT NOT NULL CHECK (unit_price > 0),
    status order_status NOT NULL DEFAULT 'PENDING',
    currency CHAR(3) NOT NULL DEFAULT 'INR'
        CHECK (currency = UPPER(currency)),
    subtotal BIGINT NOT NULL CHECK (subtotal >= 0),
    tax_amount BIGINT NOT NULL DEFAULT 0 CHECK (tax_amount >= 0),
    shipping_amount BIGINT NOT NULL DEFAULT 0 CHECK (shipping_amount >= 0),
    total_amount BIGINT NOT NULL CHECK (total_amount >= 0),
    version BIGINT NOT NULL DEFAULT 1 CHECK (version > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_orders_idempotency_key UNIQUE (idempotency_key),
    CONSTRAINT chk_orders_request_hash_hex
        CHECK (request_hash ~ '^[0-9a-fA-F]{64}$'),
    CONSTRAINT chk_orders_subtotal
        CHECK (subtotal = quantity::BIGINT * unit_price),
    CONSTRAINT chk_orders_total
        CHECK (total_amount = subtotal + tax_amount + shipping_amount)
);

CREATE TABLE order_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id UUID NOT NULL,
    quantity INTEGER NOT NULL CHECK (quantity > 0 AND quantity <= 1000),
    unit_price BIGINT NOT NULL CHECK (unit_price > 0),
    total_price BIGINT NOT NULL CHECK (total_price >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_order_items_total
        CHECK (total_price = quantity::BIGINT * unit_price)
);

CREATE TABLE order_status_history (
    id BIGSERIAL PRIMARY KEY,
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    previous_status order_status,
    new_status order_status NOT NULL,
    reason VARCHAR(500),
    changed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE idempotency_records (
    idempotency_key VARCHAR(128) PRIMARY KEY,
    request_hash CHAR(64) NOT NULL,
    state VARCHAR(20) NOT NULL
        CHECK (state IN ('PROCESSING', 'COMPLETED', 'FAILED')),
    response_status INTEGER,
    response_body JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT chk_idempotency_request_hash_hex
        CHECK (request_hash ~ '^[0-9a-fA-F]{64}$'),
    CONSTRAINT chk_idempotency_expiry
        CHECK (expires_at > created_at)
);

CREATE TABLE outbox_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    aggregate_id UUID NOT NULL,
    aggregate_type VARCHAR(50) NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    routing_key VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING'
        CHECK (status IN ('PENDING', 'PROCESSING', 'PUBLISHED', 'DEAD')),
    retry_count INTEGER NOT NULL DEFAULT 0 CHECK (retry_count >= 0),
    next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    published_at TIMESTAMPTZ
);

CREATE INDEX idx_orders_customer_created
    ON orders(customer_id, created_at DESC, id DESC);
CREATE INDEX idx_orders_status_created
    ON orders(status, created_at DESC, id DESC);
CREATE INDEX idx_orders_created
    ON orders(created_at DESC, id DESC);
CREATE INDEX idx_order_items_order
    ON order_items(order_id);
CREATE INDEX idx_order_status_history_order
    ON order_status_history(order_id, changed_at DESC);
CREATE INDEX idx_idempotency_expiry
    ON idempotency_records(expires_at);
CREATE INDEX idx_outbox_poll
    ON outbox_events(status, next_attempt_at, created_at)
    WHERE status IN ('PENDING', 'PROCESSING');
