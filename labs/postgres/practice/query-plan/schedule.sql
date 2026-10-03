-- First: ./lab.sh run query-plan. One psql session, no concurrent scenario run.
SET search_path=lab_query_plan,pg_catalog;
DROP INDEX IF EXISTS history_customer_created;
ANALYZE history;
EXPLAIN (ANALYZE,BUFFERS) SELECT id FROM history
WHERE customer_id=42 ORDER BY created_at DESC LIMIT 20;
CREATE INDEX history_customer_created ON history(customer_id,created_at DESC) INCLUDE(id);
ANALYZE history;
EXPLAIN (ANALYZE,BUFFERS) SELECT id FROM history
WHERE customer_id=42 ORDER BY created_at DESC LIMIT 20;
-- Compare actual vs estimated rows, loops, buffers, sorting and heap fetches.
-- No promise about exact milliseconds or a mandatory scan type.
