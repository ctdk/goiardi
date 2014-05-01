-- Verify roles

BEGIN;

SELECT id, name, description, run_list, env_run_lists, default_attr, override_attr, created_at, updated_at FROM roles WHERE 0;

ROLLBACK;
