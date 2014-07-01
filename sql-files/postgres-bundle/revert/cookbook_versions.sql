-- Revert cookbook_versions

BEGIN;

DROP TABLE goiardi.cookbook_versions;

COMMIT;
