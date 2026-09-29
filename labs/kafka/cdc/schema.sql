-- #region cdc-orders
CREATE TABLE IF NOT EXISTS cdc_orders (
    order_id text PRIMARY KEY,
    amount_cents bigint NOT NULL CHECK (amount_cents > 0),
    order_version bigint NOT NULL CHECK (order_version > 0)
);
-- #endregion cdc-orders

-- #region cdc-commands
CREATE TABLE IF NOT EXISTS cdc_commands (
    event_id text PRIMARY KEY,
    fingerprint text NOT NULL,
    result jsonb NOT NULL
);
-- #endregion cdc-commands

-- #region cdc-outbox
CREATE TABLE IF NOT EXISTS cdc_outbox (
    id text PRIMARY KEY,
    aggregatetype text NOT NULL CHECK (aggregatetype = 'orders'),
    aggregateid text NOT NULL,
    type text NOT NULL,
    payload jsonb NOT NULL
);
-- #endregion cdc-outbox

-- #region cdc-inbox
CREATE TABLE IF NOT EXISTS cdc_inbox (
    projection text NOT NULL,
    event_id text NOT NULL,
    fingerprint text NOT NULL,
    PRIMARY KEY (projection, event_id)
);
-- #endregion cdc-inbox

-- #region cdc-projections
CREATE TABLE IF NOT EXISTS cdc_projections (
    projection text NOT NULL,
    order_id text NOT NULL,
    amount_cents bigint NOT NULL CHECK (amount_cents > 0),
    order_version bigint NOT NULL CHECK (order_version > 0),
    PRIMARY KEY (projection, order_id)
);
-- #endregion cdc-projections
