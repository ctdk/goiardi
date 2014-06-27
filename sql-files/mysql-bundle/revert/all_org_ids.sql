-- Revert all_org_ids

BEGIN;

ALTER TABLE cookbooks DROP INDEX organization_name, ADD UNIQUE INDEX (name), DROP COLUMN organization_id;
ALTER TABLE data_bags DROP INDEX organization_name, ADD UNIQUE INDEX (name), DROP COLUMN organization_id;
ALTER TABLE environments DROP INDEX organization_name, ADD UNIQUE INDEX (name), DROP COLUMN organization_id;
ALTER TABLE nodes DROP INDEX organization_name, ADD UNIQUE INDEX (name), DROP COLUMN organization_id;
ALTER TABLE roles DROP INDEX organization_name, ADD UNIQUE INDEX (name), DROP COLUMN organization_id;
ALTER TABLE sandboxes DROP INDEX organization_sbox, ADD UNIQUE INDEX (sbox_id), DROP COLUMN organization_id;


COMMIT;
