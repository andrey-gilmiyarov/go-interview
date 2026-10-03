-- First: ./lab.sh run isolation, then open TWO ./lab.sh psql sessions.
-- Both sessions:
SET search_path=lab_isolation,pg_catalog;
-- Setup in A, outside a transaction; B must be idle:
UPDATE products SET stock=1;
-- A:
BEGIN ISOLATION LEVEL REPEATABLE READ;
SELECT sum(stock) FROM products;
-- B:
BEGIN ISOLATION LEVEL REPEATABLE READ;
SELECT sum(stock) FROM products;
-- A:
UPDATE products SET stock=0 WHERE id=1;
-- B:
UPDATE products SET stock=0 WHERE id=2;
-- A, then B:
COMMIT;
-- A:
SELECT sum(stock) FROM products;
-- Repeat after setup with SERIALIZABLE in BOTH sessions.
-- One UPDATE or COMMIT may fail with 40001. ROLLBACK the failed transaction;
-- retry the whole read/decision/write, never just the UPDATE.
