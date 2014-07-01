-- Verify data_bag_item_insert

BEGIN;

SELECT goiardi.merge_data_bags('a');
SELECT goiardi.insert_dbi('a', 'a2', 'a2', (select id from goiardi.data_bags where name = 'a'), NULL);
SELECT id FROM goiardi.data_bag_items WHERE orig_name = 'a2';

ROLLBACK;
