-- First: ./lab.sh run long-transaction; open A and B.
-- Both:
SET search_path=lab_long_transaction,pg_catalog;
-- A setup, autocommit:
UPDATE versions SET value=0;
VACUUM versions;
-- A:
BEGIN ISOLATION LEVEL REPEATABLE READ;
SELECT sum(value) FROM versions;
-- B:
UPDATE versions SET value=1;
VACUUM (VERBOSE,ANALYZE) versions;
-- A still reads old versions:
SELECT sum(value) FROM versions;
COMMIT;
-- B:
VACUUM (VERBOSE,ANALYZE) versions;
SELECT sum(value) FROM versions;
-- The relation file need not shrink. Compare visibility and VERBOSE reports.
