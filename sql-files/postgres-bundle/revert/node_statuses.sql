-- Revert node_statuses

BEGIN;

DROP TABLE goiardi.node_statuses;
DROP TYPE goiardi.status_node;

COMMIT;
