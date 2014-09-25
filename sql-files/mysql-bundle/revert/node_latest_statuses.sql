-- Revert node_latest_statuses

BEGIN;

DROP VIEW node_latest_statuses;

COMMIT;
