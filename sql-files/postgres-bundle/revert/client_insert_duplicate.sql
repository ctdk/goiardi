-- Revert client_insert_duplicate

BEGIN;

DROP FUNCTION goiardi.merge_clients(m_name text, m_nodename text, m_validator boolean, m_admin boolean, m_public_key text, m_certificate text);

COMMIT;
