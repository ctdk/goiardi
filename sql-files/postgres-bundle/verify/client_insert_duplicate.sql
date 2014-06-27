-- Verify client_insert_duplicate

BEGIN;

SELECT goiardi.merge_clients('foom', 'foom', false, false, 'asdfas', '');
SELECT id, name FROM goiardi.clients WHERE name = 'foom' AND admin = FALSE;
SELECT goiardi.merge_clients('foom', 'foom', false, true, 'asdfas', '');
SELECT id, name FROM goiardi.clients WHERE name = 'foom' AND admin = TRUE;

ROLLBACK;
