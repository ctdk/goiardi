-- Deploy sandbox_insert_update
-- requires: sandboxes
-- requires: goiardi_schema

BEGIN;

CREATE OR REPLACE FUNCTION goiardi.merge_sandboxes(m_sbox_id varchar(32), m_creation_time timestamp with time zone, m_checksums bytea, m_completed boolean) RETURNS VOID AS
$$
BEGIN
    LOOP
        -- first try to update the key
	UPDATE goiardi.sandboxes SET checksums = m_checksums, completed = m_completed WHERE sbox_id = m_sbox_id;
	IF found THEN
	    RETURN;
	END IF;
        -- not there, so try to insert the key
        -- if someone else inserts the same key concurrently,
        -- we could get a unique-key failure
        BEGIN
	    INSERT INTO goiardi.sandboxes (sbox_id, creation_time, checksums, completed) VALUES (m_sbox_id, m_creation_time, m_checksums, m_completed);
            RETURN;
        EXCEPTION WHEN unique_violation THEN
            -- Do nothing, and loop to try the UPDATE again.
        END;
    END LOOP;
END;
$$
LANGUAGE plpgsql;

COMMIT;
