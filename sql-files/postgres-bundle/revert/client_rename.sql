-- Revert client_rename

BEGIN;

DROP FUNCTION goiardi.rename_client(old_name text, new_name text);

COMMIT;
