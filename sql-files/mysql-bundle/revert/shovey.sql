-- Revert shovey

BEGIN;

DROP TABLE shovey_run_streams;
DROP TABLE shovey_runs;
DROP TABLE shoveys;

COMMIT;
