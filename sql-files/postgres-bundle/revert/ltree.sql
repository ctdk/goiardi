-- Revert ltree

BEGIN;

DROP TABLE goiardi.search_items;
DROP TABLE goiardi.search_collections;
DROP EXTENSION pg_trgm;
DROP EXTENSION ltree;

COMMIT;
