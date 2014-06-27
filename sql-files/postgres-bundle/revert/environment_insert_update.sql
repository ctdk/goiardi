-- Revert environment_insert_update

BEGIN;

DROP FUNCTION goiardi.merge_environments(m_name text, m_description text, m_default_attr bytea, m_override_attr bytea, m_cookbook_vers bytea);

COMMIT;
