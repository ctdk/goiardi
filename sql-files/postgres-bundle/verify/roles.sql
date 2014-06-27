-- Verify roles

BEGIN;

SELECT id, name, organization_id, description, run_list, env_run_lists, default_attr, override_attr, created_at, updated_at FROM goiardi.roles WHERE FALSE;

ROLLBACK;
