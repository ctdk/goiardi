-- Verify data_bags

BEGIN;

SELECT id, name, created_at, updated_at FROM data_bags WHERE 0;

ROLLBACK;
