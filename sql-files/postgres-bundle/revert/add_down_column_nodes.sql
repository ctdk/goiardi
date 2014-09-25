-- Revert add_down_column_nodes

BEGIN;

ALTER TABLE goiardi.nodes DROP COLUMN is_down;

COMMIT;
