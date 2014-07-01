-- Deploy sandboxes
-- requires: goiardi_schema

BEGIN;

CREATE TABLE goiardi.sandboxes (
	id bigserial,
	sbox_id varchar(32) not null,
	organization_id bigint not null default 1,
	creation_time timestamp with time zone not null,
	checksums bytea,
	completed boolean,
	primary key(id),
	unique(organization_id, sbox_id)
);

ALTER TABLE goiardi.sandboxes ALTER checksums SET STORAGE EXTERNAL;

COMMIT;
