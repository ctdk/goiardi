-- Verify shovey_insert_update

BEGIN;

SELECT goiardi.merge_shoveys('7c160544-460f-444f-bdbd-3f51f26bd006', 'moo', 'running', 10000, '100%');
SELECT id FROM goiardi.shoveys WHERE run_id = '7c160544-460f-444f-bdbd-3f51f26bd006';
SELECT goiardi.merge_shovey_runs('7c160544-460f-444f-bdbd-3f51f26bd006', 'moomer', 'running', NOW(), NOW(), 'werp', 0);
SELECT id FROM goiardi.shovey_runs WHERE shovey_uuid = '7c160544-460f-444f-bdbd-3f51f26bd006' AND node_name = 'moomer';

ROLLBACK;
