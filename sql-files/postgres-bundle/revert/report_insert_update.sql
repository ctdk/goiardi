-- Revert report_insert_update

BEGIN;

DROP FUNCTION goiardi.merge_reports(m_run_id uuid, m_node_name text, m_start_time timestamp with time zone, m_end_time timestamp with time zone, m_total_res_count int, m_status goiardi.report_status, m_run_list text, m_resources bytea, m_data bytea);

COMMIT;
