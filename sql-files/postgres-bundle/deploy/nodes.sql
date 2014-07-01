-- Deploy nodes
-- requires: goiardi_schema

BEGIN;

CREATE TABLE goiardi.nodes (
	id bigserial,
	name text not null,
	organization_id bigint not null default 1,
	chef_environment text not null default '_default', 
	run_list bytea,
	automatic_attr bytea,
	normal_attr bytea,
	default_attr bytea,
	override_attr bytea,
	created_at timestamp with time zone not null,
	updated_at timestamp with time zone not null,
	PRIMARY KEY(id),
	UNIQUE(organization_id, name)
);

CREATE INDEX nodes_chef_env ON goiardi.nodes(chef_environment);
ALTER TABLE goiardi.nodes ALTER run_list SET STORAGE EXTERNAL;
ALTER TABLE goiardi.nodes ALTER automatic_attr SET STORAGE EXTERNAL;
ALTER TABLE goiardi.nodes ALTER normal_attr SET STORAGE EXTERNAL;
ALTER TABLE goiardi.nodes ALTER default_attr SET STORAGE EXTERNAL;
ALTER TABLE goiardi.nodes ALTER override_attr SET STORAGE EXTERNAL;

COMMIT;
