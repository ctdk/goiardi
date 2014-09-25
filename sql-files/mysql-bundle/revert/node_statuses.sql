-- Revert node_statuses

BEGIN;

DROP TABLE node_statuses;

COMMIT;
