-- Revert role_insert_update

BEGIN;

DROP FUNCTION goiardi.merge_roles(m_name text, m_description text, m_run_list bytea, m_env_run_lists bytea, m_default_attr bytea, m_override_attr bytea);

COMMIT;
