-- Revert data_bag_item_insert

BEGIN;

DROP FUNCTION goiardi.insert_dbi(m_data_bag_name text, m_name text, m_orig_name text, m_dbag_id bigint, m_raw_data bytea);

COMMIT;
