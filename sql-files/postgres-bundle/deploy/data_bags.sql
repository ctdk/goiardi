-- Deploy data_bags
-- requires: goiardi_schema

BEGIN;

CREATE TABLE goiardi.data_bags (
	id bigserial,
	name text not null,
	organization_id bigint not null default 1,
	created_at timestamp with time zone not null,
	updated_at timestamp with time zone not null,
	primary key(id),
	unique(organization_id, name)
);

COMMIT;
