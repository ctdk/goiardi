-- Verify organizations

BEGIN;

SELECT id, name, description FROM organizations WHERE 0;

ROLLBACK;
