-- Revert file_checksums

BEGIN;

DROP TABLE goiardi.file_checksums;

COMMIT;
