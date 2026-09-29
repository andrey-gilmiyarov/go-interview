CREATE TABLE IF NOT EXISTS orders (
    order_id text PRIMARY KEY,
    amount_cents bigint NOT NULL CHECK (amount_cents > 0),
    created_at timestamptz NOT NULL DEFAULT clock_timestamp()
);

CREATE TABLE IF NOT EXISTS outbox (
    event_id text PRIMARY KEY,
    order_id text NOT NULL,
    payload jsonb NOT NULL,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    published_at timestamptz
);

CREATE INDEX IF NOT EXISTS outbox_pending_order_idx
    ON outbox (created_at, event_id)
    WHERE published_at IS NULL;

CREATE TABLE IF NOT EXISTS processed_events (
    consumer_id text NOT NULL,
    event_id text NOT NULL,
    processed_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    PRIMARY KEY (consumer_id, event_id)
);

CREATE TABLE IF NOT EXISTS order_totals (
    order_id text PRIMARY KEY,
    total_cents bigint NOT NULL CHECK (total_cents >= 0),
    updated_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
