-- Revert sandbox_insert_update

BEGIN;

DROP FUNCTION goiardi.merge_sandboxes(m_sbox_id varchar(32), m_creation_time timestamp with time zone, m_checksums bytea, m_completed boolean);

COMMIT;
