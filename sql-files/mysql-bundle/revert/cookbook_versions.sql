-- Revert cookbook_versions

BEGIN;

DROP TABLE cookbook_versions;

COMMIT;
