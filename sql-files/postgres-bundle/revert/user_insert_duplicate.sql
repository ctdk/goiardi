-- Revert user_insert_duplicate

BEGIN;

DROP FUNCTION goiardi.merge_users(m_name text, m_displayname text, m_email text, m_admin boolean, m_public_key text, m_passwd varchar(128), m_salt bytea, m_organization_id bigint);

COMMIT;
