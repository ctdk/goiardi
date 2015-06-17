-- Verify ltree_del_col

BEGIN;

INSERT INTO goiardi.search_collections (name, organization_id) VALUES ('foo', 1);
INSERT INTO goiardi.search_items (organization_id, search_collection_id, item_name, value, path) VALUES (1, (SELECT id FROM goiardi.search_collections WHERE name = 'foo' and organization_id = 1), 'beep', 'baz', 'hoo.moo.noo');
SELECT goiardi.delete_search_collection('foo', 1);

ROLLBACK;
