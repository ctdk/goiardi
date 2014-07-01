-- Deploy roles
-- requires: goiardi_schema

BEGIN;

CREATE TABLE goiardi.roles (
	id bigserial,
	name text not null,
	organization_id bigint not null default 1,
	description text,
	run_list bytea,
	env_run_lists bytea,
	default_attr bytea,
	override_attr bytea,
	created_at timestamp with time zone not null,
	updated_at timestamp with time zone not null,
	primary key(id),
	unique(organization_id, name)
);

ALTER TABLE goiardi.roles ALTER run_list SET STORAGE EXTERNAL;
ALTER TABLE goiardi.roles ALTER env_run_lists SET STORAGE EXTERNAL;
ALTER TABLE goiardi.roles ALTER default_attr SET STORAGE EXTERNAL;
ALTER TABLE goiardi.roles ALTER override_attr SET STORAGE EXTERNAL;

COMMIT;
