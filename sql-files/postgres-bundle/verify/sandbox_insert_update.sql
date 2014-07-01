-- Verify sandbox_insert_update

BEGIN;

SELECT goiardi.merge_sandboxes('moop', NOW(), NULL, FALSE);
SELECT id FROM goiardi.sandboxes WHERE sbox_id = 'moop' AND completed = FALSE;
SELECT goiardi.merge_sandboxes('moop', NOW(), NULL, TRUE);
SELECT id FROM goiardi.sandboxes WHERE sbox_id = 'moop' AND completed = TRUE;

ROLLBACK;
