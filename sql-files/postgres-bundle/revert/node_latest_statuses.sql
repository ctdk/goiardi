-- Revert node_latest_statuses

BEGIN;

DROP VIEW goiardi.node_latest_statuses;

COMMIT;
