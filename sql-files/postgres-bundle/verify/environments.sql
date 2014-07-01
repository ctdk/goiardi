-- Verify environments

BEGIN;

SELECT id, name, organization_id, description, default_attr, override_attr, cookbook_vers, created_at, updated_at FROM goiardi.environments WHERE FALSE;

ROLLBACK;
