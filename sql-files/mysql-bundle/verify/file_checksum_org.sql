-- Verify file_checksum_org

BEGIN;

SELECT id, organization_id, checksum FROM file_checksums WHERE 0;

ROLLBACK;
