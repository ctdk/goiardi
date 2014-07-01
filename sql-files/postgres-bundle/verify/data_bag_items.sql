-- Verify data_bag_items

BEGIN;

SELECT id, name, data_bag_id, raw_data, created_at, updated_at FROM goiardi.data_bag_items WHERE FALSE;

ROLLBACK;
