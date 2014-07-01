-- Verify organizations

BEGIN;

SELECT id, name, description FROM goiardi.organizations WHERE FALSE;

ROLLBACK;
