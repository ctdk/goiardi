-- Verify file_checksum_insert_ignore

BEGIN;

INSERT INTO goiardi.file_checksums (checksum) VALUES ('22871f3c541841d00ff4b7e41d176012');
INSERT INTO goiardi.file_checksums (checksum) VALUES ('22871f3c541841d00ff4b7e41d176012');

ROLLBACK;
