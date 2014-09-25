-- Verify node_status_insert

BEGIN;

SELECT goiardi.merge_nodes('moop', '_default', NULL, NULL, NULL, NULL, NULL);
SELECT goiardi.insert_node_status('moop', 'up');
SELECT id FROM goiardi.node_statuses WHERE status = 'up';

ROLLBACK;
