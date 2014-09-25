-- Verify node_statuses

BEGIN;

SELECT id, node_id, status, updated_at FROM node_statuses WHERE 0;

ROLLBACK;
