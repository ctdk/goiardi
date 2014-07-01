-- Deploy environment_insert_update
-- requires: environments
-- requires: goiardi_schema

BEGIN;

CREATE OR REPLACE FUNCTION goiardi.merge_environments(m_name text, m_description text, m_default_attr bytea, m_override_attr bytea, m_cookbook_vers bytea) RETURNS VOID AS
$$
BEGIN
    LOOP
        -- first try to update the key
	UPDATE goiardi.environments SET description = m_description, default_attr = m_default_attr, override_attr = m_override_attr, cookbook_vers = m_cookbook_vers, updated_at = NOW() WHERE name = m_name;
	IF found THEN
		RETURN;
	END IF;
        -- not there, so try to insert the key
        -- if someone else inserts the same key concurrently,
        -- we could get a unique-key failure
        BEGIN
	    INSERT INTO goiardi.environments (name, description, default_attr, override_attr, cookbook_vers, created_at, updated_at) VALUES (m_name, m_description, m_default_attr, m_override_attr, m_cookbook_vers, NOW(), NOW());
            RETURN;
        EXCEPTION WHEN unique_violation THEN
            -- Do nothing, and loop to try the UPDATE again.
        END;
    END LOOP;
END;
$$
LANGUAGE plpgsql;

COMMIT;
