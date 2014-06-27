-- Deploy clients
-- requires: goiardi_schema

BEGIN;

CREATE TABLE goiardi.clients (
	id bigserial,
	name text not null,
	nodename text,
	validator boolean,
	admin boolean,
	organization_id bigint not null default 1,
	public_key text,
	certificate text,
	created_at timestamp with time zone not null,
	updated_at timestamp with time zone not null,
	primary key(id),
	unique(organization_id, name)
);

COMMIT;
