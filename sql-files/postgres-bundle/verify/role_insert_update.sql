-- Verify role_insert_update

BEGIN;

SELECT goiardi.merge_roles('moop', '_default', NULL, NULL, NULL, NULL);
SELECT id FROM goiardi.roles WHERE name = 'moop' AND description = '_default';
SELECT goiardi.merge_roles('moop', 'hoopty', NULL, NULL, NULL, NULL);
SELECT id FROM goiardi.roles WHERE name = 'moop' AND description = 'hoopty';

ROLLBACK;
