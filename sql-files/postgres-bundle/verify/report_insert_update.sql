-- Verify report_insert_update

BEGIN;

SELECT goiardi.merge_reports('7c160544-460f-444f-bdbd-3f51f26bd006', 'moo', NOW(), NULL, 0, 'started', '', NULL, NULL);
SELECT id FROM goiardi.reports WHERE run_id = '7c160544-460f-444f-bdbd-3f51f26bd006' AND status = 'started';
SELECT goiardi.merge_reports('7c160544-460f-444f-bdbd-3f51f26bd006', 'moo', NOW(), NOW(), 0, 'success', '', NULL, NULL);
SELECT id FROM goiardi.reports WHERE run_id = '7c160544-460f-444f-bdbd-3f51f26bd006' AND status = 'success';

ROLLBACK;
