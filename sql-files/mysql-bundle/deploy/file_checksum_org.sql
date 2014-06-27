-- Deploy file_checksum_org
-- requires: file_checksums

BEGIN;

UPDATE file_checksums SET org_id = 1 WHERE org_id = 0;
ALTER TABLE file_checksums CHANGE COLUMN org_id organization_id INT NOT NULL DEFAULT 1;

COMMIT;
