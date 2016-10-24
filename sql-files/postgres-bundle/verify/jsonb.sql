-- Verify goiardi_postgres:jsonb on pg

BEGIN;

SELECT id, cookbook_id, major_ver, minor_ver, patch_ver, frozen, metadata, definitions, libraries, attributes, recipes, providers, resources, templates, root_files, files, created_at, updated_at FROM goiardi.cookbook_versions WHERE FALSE;

SELECT id, name, data_bag_id, raw_data, created_at, updated_at FROM goiardi.data_bag_items WHERE FALSE;

SELECT id, name, organization_id, description, default_attr, override_attr, cookbook_vers, created_at, updated_at FROM goiardi.environments WHERE FALSE;

SELECT id, name, organization_id, chef_environment, automatic_attr, normal_attr, default_attr, override_attr, created_at, updated_at FROM goiardi.nodes WHERE FALSE;

SELECT id, run_id, node_name, organization_id, start_time, end_time, total_res_count, status, run_list, resources, data, created_at, updated_at FROM goiardi.reports WHERE FALSE;

SELECT id, name, organization_id, description, run_list, env_run_lists, default_attr, override_attr, created_at, updated_at FROM goiardi.roles WHERE FALSE;

SELECT id, sbox_id, organization_id, creation_time, checksums FROM goiardi.sandboxes WHERE FALSE;

ROLLBACK;
