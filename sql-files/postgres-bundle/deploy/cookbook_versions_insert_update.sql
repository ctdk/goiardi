-- Deploy cookbook_versions_insert_update
-- requires: cookbook_versions
-- requires: goiardi_schema

BEGIN;

CREATE OR REPLACE FUNCTION goiardi.merge_cookbook_versions(c_id bigint, is_frozen bool, defb bytea, libb bytea, attb bytea, recb bytea, prob bytea, resb bytea, temb bytea, roob bytea, filb bytea, metb bytea, maj bigint, min bigint, patch bigint) RETURNS BIGINT AS
$$
DECLARE
    cbv_id BIGINT;
BEGIN
    LOOP
        -- first try to update the key
        UPDATE goiardi.cookbook_versions SET frozen = is_frozen, metadata = metb, definitions = defb, libraries = libb, attributes = attb, recipes = recb, providers = prob, resources = resb, templates = temb, root_files = roob, files = filb, updated_at = NOW() WHERE cookbook_id = c_id AND major_ver = maj AND minor_ver = min AND patch_ver = patch RETURNING id INTO cbv_id;
        IF found THEN
            RETURN cbv_id;
        END IF;
        -- not there, so try to insert the key
        -- if someone else inserts the same key concurrently,
        -- we could get a unique-key failure
        BEGIN
            INSERT INTO goiardi.cookbook_versions (cookbook_id, major_ver, minor_ver, patch_ver, frozen, metadata, definitions, libraries, attributes, recipes, providers, resources, templates, root_files, files, created_at, updated_at) VALUES (c_id, maj, min, patch, is_frozen, metb, defb, libb, attb, recb, prob, resb, temb, roob, filb, NOW(), NOW()) RETURNING id INTO cbv_id;
            RETURN c_id;
        EXCEPTION WHEN unique_violation THEN
            -- Do nothing, and loop to try the UPDATE again.
        END;
    END LOOP;
END;
$$
LANGUAGE plpgsql;

COMMIT;
