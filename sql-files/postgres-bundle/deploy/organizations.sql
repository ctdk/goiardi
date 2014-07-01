-- Deploy organizations

BEGIN;

CREATE TABLE goiardi.organizations (
	id bigserial,
	name text not null,
	description text,
	created_at timestamp with time zone not null,
	updated_at timestamp with time zone not null,
	primary key(id),
	unique(name)
);
INSERT INTO goiardi.organizations (name, created_at, updated_at) VALUES ('default', NOW(), NOW());

COMMIT;
