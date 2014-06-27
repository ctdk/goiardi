-- Verify data_bag_insert_update

BEGIN;

SELECT goiardi.merge_data_bags('moo');
SELECT updated_at INTO TEMPORARY old_dbag FROM goiardi.data_bags WHERE name = 'moo';
SELECT goiardi.merge_cookbooks('moo');
SELECT d.updated_at FROM goiardi.data_bags d, old_dbag WHERE name = 'moo' AND d.updated_at <> old_dbag.updated_at;

ROLLBACK;
