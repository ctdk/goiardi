-- Verify file_checksums

BEGIN;

SELECT id, org_id, checksum FROM file_checksums WHERE 0;

ROLLBACK;
