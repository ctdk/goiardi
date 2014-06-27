-- Verify node_insert_ignore

BEGIN;

SELECT goiardi.merge_nodes('moop', '_default', NULL, NULL, NULL, NULL, NULL);
SELECT id FROM goiardi.nodes WHERE name = 'moop' AND chef_environment = '_default';
SELECT goiardi.merge_nodes('moop', 'hoopty', NULL, NULL, NULL, NULL, NULL);
SELECT id FROM goiardi.nodes WHERE name = 'moop' AND chef_environment = 'hoopty';

ROLLBACK;
