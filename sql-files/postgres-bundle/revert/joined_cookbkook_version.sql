-- Revert joined_cookbkook_version

-- Adapted from the erchef joined_cookbook_version view, found in the repo
-- at https://github.com/opscode/chef-server-schema.

BEGIN;

DROP VIEW IF EXISTS goiardi.joined_cookbook_version;

COMMIT;
