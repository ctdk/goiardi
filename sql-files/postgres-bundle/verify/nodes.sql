-- Verify nodes

BEGIN;

SELECT id, name, organization_id, chef_environment, automatic_attr, normal_attr, default_attr, override_attr, created_at, updated_at FROM goiardi.nodes WHERE FALSE;

ROLLBACK;
