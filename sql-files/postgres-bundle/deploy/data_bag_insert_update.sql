-- Deploy data_bag_insert_update
-- requires: data_bags
-- requires: goiardi_schema

BEGIN;

CREATE OR REPLACE FUNCTION goiardi.merge_data_bags(m_name text) RETURNS BIGINT AS
$$
DECLARE
    db_id BIGINT;
BEGIN
    LOOP
        -- first try to update the key
        UPDATE goiardi.data_bags SET updated_at = NOW() WHERE name = m_name RETURNING id INTO db_id;
        IF found THEN
            RETURN db_id;
        END IF;
        -- not there, so try to insert the key
        -- if someone else inserts the same key concurrently,
        -- we could get a unique-key failure
        BEGIN
            INSERT INTO goiardi.data_bags (name, created_at, updated_at) VALUES (m_name, NOW(), NOW()) RETURNING id INTO db_id;
            RETURN db_id;
        EXCEPTION WHEN unique_violation THEN
            -- Do nothing, and loop to try the UPDATE again.
        END;
    END LOOP;
END;
$$
LANGUAGE plpgsql;

COMMIT;
