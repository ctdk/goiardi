-- Deploy role_insert_update
-- requires: roles
-- requires: goiardi_schema

BEGIN;

CREATE OR REPLACE FUNCTION goiardi.merge_roles(m_name text, m_description text, m_run_list bytea, m_env_run_lists bytea, m_default_attr bytea, m_override_attr bytea) RETURNS VOID AS
$$
BEGIN
    LOOP
        -- first try to update the key
	UPDATE goiardi.roles SET description = m_description, run_list = m_run_list, env_run_lists = m_env_run_lists, default_attr = m_default_attr, override_attr = m_override_attr, updated_at = NOW() WHERE name = m_name;
	IF found THEN
	    RETURN;
	END IF;
        -- not there, so try to insert the key
        -- if someone else inserts the same key concurrently,
        -- we could get a unique-key failure
        BEGIN
	    INSERT INTO goiardi.roles (name, description, run_list, env_run_lists, default_attr, override_attr, created_at, updated_at) VALUES (m_name, m_description, m_run_list, m_env_run_lists, m_default_attr, m_override_attr, NOW(), NOW());
            RETURN;
        EXCEPTION WHEN unique_violation THEN
            -- Do nothing, and loop to try the UPDATE again.
        END;
    END LOOP;
END;
$$
LANGUAGE plpgsql;

COMMIT;
