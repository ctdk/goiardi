-- Verify cookbook_versions_insert_update

BEGIN;

SELECT goiardi.merge_cookbooks('moo');
SELECT goiardi.merge_cookbook_versions((SELECT id FROM goiardi.cookbooks WHERE name = 'moo'), false, null, null, null, null, null, null, null, null, null, null, 1, 1, 1);
SELECT id FROM goiardi.cookbook_versions WHERE cookbook_id = (SELECT id FROM goiardi.cookbooks WHERE name = 'moo') AND major_ver = 1 AND minor_ver = 1 AND patch_ver = 1 AND frozen = FALSE;
SELECT goiardi.merge_cookbook_versions((SELECT id FROM goiardi.cookbooks WHERE name = 'moo'), true, null, null, null, null, null, null, null, null, null, null, 1, 1, 1);
SELECT id FROM goiardi.cookbook_versions WHERE cookbook_id = (SELECT id FROM goiardi.cookbooks WHERE name = 'moo') AND major_ver = 1 AND minor_ver = 1 AND patch_ver = 1 AND frozen = TRUE;

ROLLBACK;
