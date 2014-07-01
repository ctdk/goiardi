-- Deploy users
-- requires: goiardi_schema

BEGIN;

CREATE TABLE goiardi.users (
	id bigserial,
	name text not null,
	displayname text,
	email text,
	admin boolean,
	public_key text,
	passwd varchar(128),
	salt bytea,
	created_at timestamp with time zone not null,
	updated_at timestamp with time zone not null,
	primary key(id),
	unique(name),
	unique(email)
);

COMMIT;
