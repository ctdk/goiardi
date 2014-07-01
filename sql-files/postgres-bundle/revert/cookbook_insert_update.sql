-- Revert cookbook_insert_update

BEGIN;

DROP FUNCTION goiardi.merge_cookbooks(m_name text);

COMMIT;
