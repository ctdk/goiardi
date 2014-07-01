-- Revert file_checksum_insert_ignore

BEGIN;

DROP RULE insert_ignore ON goiardi.file_checksums;

COMMIT;
