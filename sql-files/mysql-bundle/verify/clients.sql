-- Verify clients

BEGIN;

SELECT id, name, nodename, validator, admin, organization_id, public_key, certificate, created_at, updated_at FROM clients WHERE 0;

ROLLBACK;
