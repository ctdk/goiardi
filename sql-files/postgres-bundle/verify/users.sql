-- Verify users

BEGIN;

SELECT id, name, displayname, email, admin, public_key, passwd, salt, created_at, updated_at FROM goiardi.users WHERE FALSE;

ROLLBACK;
