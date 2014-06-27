-- Deploy node_insert_ignore
-- requires: nodes
-- requires: goiardi_schema

BEGIN;

CREATE OR REPLACE FUNCTION goiardi.merge_nodes(m_name text, m_chef_environment text, m_run_list bytea, m_automatic_attr bytea, m_normal_attr bytea, m_default_attr bytea, m_override_attr bytea) RETURNS VOID AS
$$
BEGIN
    LOOP
        -- first try to update the key
	UPDATE goiardi.nodes SET chef_environment = m_chef_environment, run_list = m_run_list, automatic_attr = m_automatic_attr, normal_attr = m_normal_attr, default_attr = m_default_attr, override_attr = m_override_attr, updated_at = NOW() WHERE name = m_name;
	IF found THEN
	    RETURN;
	END IF;
        -- not there, so try to insert the key
        -- if someone else inserts the same key concurrently,
        -- we could get a unique-key failure
        BEGIN
	    INSERT INTO goiardi.nodes (name, chef_environment, run_list, automatic_attr, normal_attr, default_attr, override_attr, created_at, updated_at) VALUES (m_name, m_chef_environment, m_run_list, m_automatic_attr, m_normal_attr, m_default_attr, m_override_attr, NOW(), NOW());
            RETURN;
        EXCEPTION WHEN unique_violation THEN
            -- Do nothing, and loop to try the UPDATE again.
        END;
    END LOOP;
END;
$$
LANGUAGE plpgsql;

COMMIT;
