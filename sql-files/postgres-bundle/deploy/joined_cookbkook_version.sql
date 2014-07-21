-- Deploy joined_cookbkook_version

-- Adapted from the erchef joined_cookbook_version view, found in the repo
-- at https://github.com/opscode/chef-server-schema.

BEGIN;

CREATE OR REPLACE VIEW goiardi.joined_cookbook_version(
    -- Cookbook Version fields
    major_ver, -- these 3 are needed for version information (duh)
    minor_ver,
    patch_ver,
    version, -- concatenated string of the complete version
    id, -- used for retrieving environment-filtered recipes
    metadata,
    recipes,
    -- Cookbook fields
    organization_id, -- not actually doing anything yet
    name) -- both version and recipe queries require the cookbook name
AS
SELECT v.major_ver,
       v.minor_ver,
       v.patch_ver,
       v.major_ver || '.' || v.minor_ver || '.' || v.patch_ver,
       v.id,
       v.metadata,
       v.recipes,
       c.organization_id,
       c.name
FROM goiardi.cookbooks AS c
JOIN goiardi.cookbook_versions AS v
  ON c.id = v.cookbook_id;

COMMIT;
