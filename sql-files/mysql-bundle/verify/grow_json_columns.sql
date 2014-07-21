-- Verify grow_json_columns

BEGIN;

SELECT id, name, chef_environment, automatic_attr, normal_attr, default_attr, override_attr, created_at, updated_at FROM nodes WHERE 0;
SELECT id, name, data_bag_id, raw_data, created_at, updated_at FROM data_bag_items WHERE 0;
SELECT id, name, description, default_attr, override_attr, cookbook_vers, created_at, updated_at FROM environments WHERE 0;
SELECT id, cookbook_id, major_ver, minor_ver, patch_ver, frozen, metadata, definitions, libraries, attributes, recipes, providers, resources, templates, root_files, files, created_at, updated_at FROM cookbook_versions WHERE 0;
SELECT id, run_id, node_name, organization_id, start_time, end_time, total_res_count, status, run_list, resources, data, created_at, updated_at FROM reports WHERE 0;
SELECT id, name, description, run_list, env_run_lists, default_attr, override_attr, created_at, updated_at FROM roles WHERE 0;
SELECT id, sbox_id, creation_time, checksums FROM sandboxes WHERE 0;

ROLLBACK;
