-- Revert data_bag_insert_update

BEGIN;

DROP FUNCTION goiardi.merge_data_bags(m_name text);

COMMIT;
