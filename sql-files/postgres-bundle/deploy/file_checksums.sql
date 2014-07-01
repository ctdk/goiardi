-- Deploy file_checksums

BEGIN;

CREATE TABLE goiardi.file_checksums (
	id bigserial,
	organization_id bigint not null default 1,
	checksum varchar(32),
	primary key(id),
	unique(organization_id, checksum)
);

COMMIT;
