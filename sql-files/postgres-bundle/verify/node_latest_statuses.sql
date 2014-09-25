-- Verify node_latest_statuses

BEGIN;

SELECT id, name, chef_environment, run_list, automatic_attr, normal_attr, default_attr, override_attr, is_down, status, updated_at FROM goiardi.node_latest_statuses WHERE false;

ROLLBACK;
