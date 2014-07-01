-- Revert node_insert_ignore

BEGIN;

DROP FUNCTION goiardi.merge_nodes(m_name text, m_chef_environment text, m_run_list bytea, m_automatic_attr bytea, m_normal_attr bytea, m_default_attr bytea, m_override_attr bytea);

COMMIT;
