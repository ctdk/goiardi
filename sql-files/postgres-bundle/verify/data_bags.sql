-- Verify data_bags

BEGIN;

SELECT id, name, organization_id, created_at, updated_at FROM goiardi.data_bags WHERE FALSE;

ROLLBACK;
