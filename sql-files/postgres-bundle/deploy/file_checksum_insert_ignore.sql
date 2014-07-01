-- Deploy file_checksum_insert_ignore
-- requires: file_checksums
-- requires: goiardi_schema

BEGIN;

CREATE RULE insert_ignore AS ON INSERT TO goiardi.file_checksums
	WHERE EXISTS(SELECT 1 FROM goiardi.file_checksums 
		WHERE (organization_id, checksum)=(NEW.organization_id, NEW.checksum))
	DO INSTEAD NOTHING;

COMMIT;
