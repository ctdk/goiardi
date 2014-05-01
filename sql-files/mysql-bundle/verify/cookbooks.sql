-- Verify cookbooks

BEGIN;

SELECT id, name, created_at, updated_at FROM cookbooks WHERE 0;

ROLLBACK;
