-- Revert data_bag_items

BEGIN;

DROP TABLE goiardi.data_bag_items;

COMMIT;
