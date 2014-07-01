-- Revert log_infos

BEGIN;

DROP TABLE goiardi.log_infos;
DROP TYPE goiardi.log_action;
DROP TYPE goiardi.log_actor;

COMMIT;
