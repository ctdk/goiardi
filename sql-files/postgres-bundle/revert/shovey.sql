-- Revert shovey

BEGIN;

DROP TABLE goiardi.shovey_run_streams;
DROP TABLE goiardi.shovey_runs;
DROP TABLE goiardi.shoveys;
DROP TYPE goiardi.shovey_output;

COMMIT;
