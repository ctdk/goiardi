-- Verify user_insert_duplicate

BEGIN;

SELECT goiardi.merge_users('foom', 'foom', '', false, 'asdfas', '', NULL, 1);
SELECT id, name FROM goiardi.users WHERE name = 'foom' AND admin = FALSE;
SELECT goiardi.merge_users('foom', 'foom', '', true, 'asdfas', '', NULL, 1);
SELECT id, name FROM goiardi.users WHERE name = 'foom' AND admin = TRUE;

ROLLBACK;
