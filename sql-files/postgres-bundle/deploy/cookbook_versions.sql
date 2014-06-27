-- Deploy cookbook_versions
-- requires: cookbooks
-- requires: goiardi_schema

BEGIN;

CREATE TABLE goiardi.cookbook_versions (
	id bigserial,
	cookbook_id bigint not null,
	major_ver bigint not null,
	minor_ver bigint not null,
	patch_ver bigint not null default 0, -- the first two *must* be set,
					     -- the third not necessarily. This
					     -- may be better as a nullable
					     -- column though.
	frozen boolean,
	metadata bytea,
	definitions bytea,
	libraries bytea,
	attributes bytea,
	recipes bytea,
	providers bytea,
	resources bytea,
	templates bytea,
	root_files bytea,
	files bytea,
	created_at timestamp with time zone not null,
	updated_at timestamp with time zone not null,
	PRIMARY KEY(id),
	UNIQUE(cookbook_id, major_ver, minor_ver, patch_ver),
	FOREIGN KEY (cookbook_id)
		REFERENCES goiardi.cookbooks(id)
		ON DELETE RESTRICT
);

COMMIT;
