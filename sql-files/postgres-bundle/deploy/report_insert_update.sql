-- Deploy report_insert_update
-- requires: reports
-- requires: goiardi_schema

BEGIN;

CREATE OR REPLACE FUNCTION goiardi.merge_reports(m_run_id uuid, m_node_name text, m_start_time timestamp with time zone, m_end_time timestamp with time zone, m_total_res_count int, m_status goiardi.report_status, m_run_list text, m_resources bytea, m_data bytea) RETURNS VOID AS
$$
BEGIN
    LOOP
        -- first try to update the key
	UPDATE goiardi.reports SET start_time = m_start_time, end_time = m_end_time, total_res_count = m_total_res_count, status = m_status, run_list = m_run_list, resources = m_resources, data = m_data, updated_at = NOW() WHERE run_id = m_run_id;
	IF found THEN
	    RETURN;
	END IF;
        -- not there, so try to insert the key
        -- if someone else inserts the same key concurrently,
        -- we could get a unique-key failure
        BEGIN
	    INSERT INTO goiardi.reports (run_id, node_name, start_time, end_time, total_res_count, status, run_list, resources, data, created_at, updated_at) VALUES (m_run_id, m_node_name, m_start_time, m_end_time, m_total_res_count, m_status, m_run_list, m_resources, m_data, NOW(), NOW());
            RETURN;
        EXCEPTION WHEN unique_violation THEN
            -- Do nothing, and loop to try the UPDATE again.
        END;
    END LOOP;
END;
$$
LANGUAGE plpgsql;

COMMIT;
