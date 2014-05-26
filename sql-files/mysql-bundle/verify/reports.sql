-- Verify reports

BEGIN;

SELECT id, run_id, node_name, organization_id, start_time, end_time, total_res_count, status, run_list, resources, data, created_at, updated_at FROM reports WHERE 0;

ROLLBACK;
