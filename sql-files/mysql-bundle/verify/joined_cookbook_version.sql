-- Verify joined_cookbook_version

BEGIN;

SELECT major_ver, minor_ver, patch_ver, version, metadata, recipes, id, organization_id, name FROM joined_cookbook_version WHERE 0;

ROLLBACK;
