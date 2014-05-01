-- Verify cookbook_versions

BEGIN;

SELECT id, cookbook_id, major_ver, minor_ver, patch_ver, frozen, metadata, definitions, libraries, attributes, recipes, providers, resources, templates, root_files, files, created_at, updated_at FROM cookbook_versions WHERE 0;

ROLLBACK;
