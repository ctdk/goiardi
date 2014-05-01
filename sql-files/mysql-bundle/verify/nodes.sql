-- Verify nodes

BEGIN;

SELECT id, name, chef_environment, automatic_attr, normal_attr, default_attr, override_attr, created_at, updated_at FROM nodes WHERE 0;

ROLLBACK;
