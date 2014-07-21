-- Deploy grow_json_columns

BEGIN;

ALTER TABLE cookbook_versions 
	MODIFY metadata mediumtext,
	MODIFY definitions mediumtext,
	MODIFY libraries mediumtext,
	MODIFY attributes mediumtext,
	MODIFY recipes mediumtext,
	MODIFY providers mediumtext,
	MODIFY resources mediumtext,
	MODIFY templates mediumtext,
	MODIFY root_files mediumtext,
	MODIFY files mediumtext;

ALTER TABLE data_bag_items
	MODIFY raw_data mediumtext;

ALTER TABLE environments
	MODIFY default_attr mediumtext,
	MODIFY override_attr mediumtext,
	MODIFY cookbook_vers mediumtext;

ALTER TABLE nodes
	MODIFY run_list mediumtext,
	MODIFY automatic_attr mediumtext,
	MODIFY normal_attr mediumtext,
	MODIFY default_attr mediumtext,
	MODIFY override_attr mediumtext;

ALTER TABLE reports
	MODIFY resources mediumtext,
	MODIFY data mediumtext;

ALTER TABLE roles
	MODIFY run_list mediumtext,
	MODIFY env_run_lists mediumtext,
	MODIFY default_attr mediumtext,
	MODIFY override_attr mediumtext;

ALTER TABLE sandboxes
	MODIFY checksums mediumtext;

COMMIT;
