-- First: ./lab.sh run deadlock. Open A, B and optional observer O.
-- All sessions:
SET search_path=lab_deadlock,pg_catalog;
-- A:
BEGIN;
UPDATE products SET stock=stock WHERE id=1;
-- B:
BEGIN;
UPDATE products SET stock=stock WHERE id=2;
-- A (blocks; switch to B without waiting for completion):
UPDATE products SET stock=stock WHERE id=2;
-- Observer:
SELECT pid,pg_blocking_pids(pid) FROM pg_stat_activity WHERE datname=current_database();
-- B (creates cycle):
UPDATE products SET stock=stock WHERE id=1;
-- BOTH sessions, after detector resolves the cycle:
ROLLBACK;
-- Repeat with 1 then 2 in both sessions; commit A so B can proceed.
