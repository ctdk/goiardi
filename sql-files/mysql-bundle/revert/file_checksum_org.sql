-- Revert file_checksum_org

BEGIN;

ALTER TABLE file_checksums CHANGE COLUMN organization_id org_id INT NOT NULL DEFAULT 0;
UPDATE file_checksums SET org_id = 0 WHERE org_id = 1;

COMMIT;
