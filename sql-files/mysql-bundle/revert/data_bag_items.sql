-- Revert data_bag_items

BEGIN;

DROP TABLE data_bag_items;

COMMIT;
