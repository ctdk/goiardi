-- Verify goiardi_schema

BEGIN;

SELECT pg_catalog.has_schema_privilege('goiardi', 'usage');

ROLLBACK;
