-- Verify sandboxes

BEGIN;

SELECT id, sbox_id, creation_time, checksums FROM sandboxes WHERE 0;

ROLLBACK;
