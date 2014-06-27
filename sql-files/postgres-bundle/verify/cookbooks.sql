-- Verify cookbooks

BEGIN;

SELECT id, name, organization_id, created_at, updated_at FROM goiardi.cookbooks WHERE FALSE;

ROLLBACK;
