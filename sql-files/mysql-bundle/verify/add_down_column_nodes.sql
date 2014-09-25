-- Verify add_down_column_nodes

BEGIN;

SELECT is_down FROM nodes WHERE 0;

ROLLBACK;
