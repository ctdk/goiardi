-- Verify shovey

BEGIN;

SELECT id, run_id, command, status, timeout, quorum, created_at, updated_at, organization_id FROM goiardi.shoveys WHERE false;
SELECT id, shovey_uuid, shovey_id, node_name, status, ack_time, end_time, error, exit_status FROM goiardi.shovey_runs WHERE false;
SELECT id, shovey_run_id, seq, output_type, output, is_last, created_at FROM goiardi.shovey_run_streams WHERE false;

ROLLBACK;
