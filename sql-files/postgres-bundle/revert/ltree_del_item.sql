-- Revert ltree_del_item

BEGIN;

DROP FUNCTION goiardi.delete_search_item(col text, item text, m_organization_id int);

COMMIT;
