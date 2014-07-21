-- Verify joined_cookbkook_version
-- Adapted from the erchef joined_cookbook_version view, found in the repo
-- at https://github.com/opscode/chef-server-schema.

BEGIN;

SELECT major_ver, minor_ver, patch_ver, version, metadata, recipes, id, organization_id, name FROM goiardi.joined_cookbook_version WHERE FALSE;

ROLLBACK;
