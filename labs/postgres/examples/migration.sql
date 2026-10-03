-- Only run in a lab-owned schema, in psql autocommit mode.
-- Expanded schema supports old writers until a deliberate cutover.
SET lock_timeout = '2s';
CREATE TABLE migration_orders(id bigint PRIMARY KEY, customer_id bigint);
INSERT INTO migration_orders SELECT n,NULL FROM generate_series(1,1000)n;
-- Real backfills repeat bounded primary-key ranges and coordinate new writes.
UPDATE migration_orders SET customer_id=id WHERE id BETWEEN 1 AND 1000;
ALTER TABLE migration_orders ADD CONSTRAINT customer_present CHECK(customer_id IS NOT NULL) NOT VALID;
ALTER TABLE migration_orders VALIDATE CONSTRAINT customer_present;
CREATE INDEX CONCURRENTLY migration_orders_customer ON migration_orders(customer_id);
-- Execute only after every writer supplies customer_id.
ALTER TABLE migration_orders ALTER COLUMN customer_id SET NOT NULL;
RESET lock_timeout;
