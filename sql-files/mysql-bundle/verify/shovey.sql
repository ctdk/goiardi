-- Verify shovey

BEGIN;

SELECT id, run_id, command, status, timeout, quorum, created_at, updated_at, organization_id FROM shoveys WHERE 0;
SELECT id, shovey_uuid, shovey_id, node_name, status, ack_time, end_time, error, exit_status FROM shovey_runs WHERE 0;
SELECT id, shovey_run_id, seq, output_type, output, is_last, created_at FROM shovey_run_streams WHERE 0;

ROLLBACK;
