-- Revert grow_json_columns

BEGIN;

ALTER TABLE cookbook_versions 
	MODIFY metadata blob,
	MODIFY definitions blob,
	MODIFY libraries blob,
	MODIFY attributes blob,
	MODIFY recipes blob,
	MODIFY providers blob,
	MODIFY resources blob,
	MODIFY templates blob,
	MODIFY root_files blob,
	MODIFY files blob;

ALTER TABLE data_bag_items
	MODIFY raw_data blob;

ALTER TABLE environments
	MODIFY default_attr blob,
	MODIFY override_attr blob,
	MODIFY cookbook_vers blob;

ALTER TABLE nodes
	MODIFY run_list blob,
	MODIFY automatic_attr blob,
	MODIFY normal_attr blob,
	MODIFY default_attr blob,
	MODIFY override_attr blob;

ALTER TABLE reports
	MODIFY resources blob,
	MODIFY data blob;

ALTER TABLE roles
	MODIFY run_list blob,
	MODIFY env_run_lists blob,
	MODIFY default_attr blob,
	MODIFY override_attr blob;

ALTER TABLE sandboxes
	MODIFY checksums blob;

COMMIT;
