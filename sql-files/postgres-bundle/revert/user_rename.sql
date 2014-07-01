-- Revert user_rename

BEGIN;

DROP FUNCTION goiardi.rename_user(old_name text, new_name text, m_organization_id int);

COMMIT;
