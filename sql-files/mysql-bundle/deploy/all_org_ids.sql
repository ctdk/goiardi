-- Deploy all_org_ids

BEGIN;

ALTER TABLE cookbooks ADD COLUMN organization_id INT NOT NULL DEFAULT 1, DROP INDEX name, ADD UNIQUE INDEX organization_name (organization_id, name);
ALTER TABLE data_bags ADD COLUMN organization_id INT NOT NULL DEFAULT 1, DROP INDEX name, ADD UNIQUE INDEX organization_name (organization_id, name);
ALTER TABLE environments ADD COLUMN organization_id INT NOT NULL DEFAULT 1, DROP INDEX name, ADD UNIQUE INDEX organization_name (organization_id, name);
ALTER TABLE nodes ADD COLUMN organization_id INT NOT NULL DEFAULT 1, DROP INDEX name, ADD UNIQUE INDEX organization_name (organization_id, name);
ALTER TABLE roles ADD COLUMN organization_id INT NOT NULL DEFAULT 1, DROP INDEX name, ADD UNIQUE INDEX organization_name (organization_id, name);
ALTER TABLE sandboxes ADD COLUMN organization_id INT NOT NULL DEFAULT 1, DROP INDEX sbox_id, ADD UNIQUE INDEX organization_sbox (organization_id, sbox_id);

COMMIT;
