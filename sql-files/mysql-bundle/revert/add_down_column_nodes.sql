-- Revert add_down_column_nodes

BEGIN;

ALTER TABLE nodes DROP COLUMN is_down;

COMMIT;
