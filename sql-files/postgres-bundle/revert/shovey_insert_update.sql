-- Revert shovey_insert_update

BEGIN;

DROP FUNCTION goiardi.merge_shoveys(m_run_id uuid, m_command text, m_status text, m_timeout bigint, m_quorum varchar(25));
DROP FUNCTION goiardi.merge_shovey_runs(m_shovey_run_id uuid, m_node_name text, m_status text, m_ack_time timestamp with time zone, m_end_time timestamp with time zone, m_error text, m_exit_status integer);

COMMIT;
