-- Verify all_org_ids

BEGIN;

SELECT organization_id FROM cookbooks WHERE 1;
SELECT organization_id FROM data_bags WHERE 1;
SELECT organization_id FROM environments WHERE 1;
SELECT organization_id FROM nodes WHERE 1;
SELECT organization_id FROM roles WHERE 1;
SELECT organization_id FROM sandboxes WHERE 1;

ROLLBACK;
