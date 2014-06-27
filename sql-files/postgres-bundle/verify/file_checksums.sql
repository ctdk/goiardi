-- Verify file_checksums

BEGIN;

SELECT id, organization_id, checksum FROM goiardi.file_checksums WHERE FALSE;

ROLLBACK;
