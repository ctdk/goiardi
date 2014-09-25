-- Verify node_statuses

BEGIN;

SELECT id, node_id, status, updated_at FROM goiardi.node_statuses WHERE FALSE;

ROLLBACK;
