-- Deploy joined_cookbook_version

BEGIN;

CREATE OR REPLACE VIEW joined_cookbook_version(
    -- Cookbook Version fields
    major_ver, -- these 3 are needed for version information (duh)
    minor_ver,
    patch_ver,
    version, -- concatenated string of the complete version
    id, -- used for retrieving environment-filtered recipes
    metadata,
    recipes,
    -- Cookbook fields
    organization_id,
    name) -- both version and recipe queries require the cookbook name
AS
SELECT v.major_ver,
       v.minor_ver,
       v.patch_ver,
       concat(v.major_ver, '.', v.minor_ver, '.', v.patch_ver),
       v.id,
       v.metadata,
       v.recipes,
       c.organization_id,
       c.name
FROM cookbooks AS c
JOIN cookbook_versions AS v
  ON c.id = v.cookbook_id;

COMMIT;
