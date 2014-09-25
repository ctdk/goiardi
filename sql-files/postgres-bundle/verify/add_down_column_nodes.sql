-- Verify add_down_column_nodes

BEGIN;

SELECT is_down FROM goiardi.nodes WHERE false;

ROLLBACK;
