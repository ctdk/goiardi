-- Revert reports

BEGIN;

DROP TABLE goiardi.reports;
DROP TYPE goiardi.report_status;

COMMIT;
