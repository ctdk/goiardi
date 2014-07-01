-- Deploy environments
-- requires: goiardi_schema

BEGIN;

CREATE TABLE goiardi.environments (
	id bigserial,
	name text,
	organization_id bigint not null default 1,
	description text,
	default_attr bytea,
	override_attr bytea,
	cookbook_vers bytea, -- make a blob for now, may bust out to a table
	created_at timestamp with time zone not null,
	updated_at timestamp with time zone not null,
	PRIMARY KEY(id),
	UNIQUE(organization_id, name)
);
ALTER TABLE goiardi.environments ALTER default_attr SET STORAGE EXTERNAL;
ALTER TABLE goiardi.environments ALTER override_attr SET STORAGE EXTERNAL;
ALTER TABLE goiardi.environments ALTER cookbook_vers SET STORAGE EXTERNAL;

INSERT INTO goiardi.environments (id, name, description, created_at, updated_at) VALUES (1, '_default', 'The default Chef environment', NOW(), NOW());

COMMIT;
