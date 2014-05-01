-- Revert file_checksums

BEGIN;

DROP TABLE file_checksums;

COMMIT;
