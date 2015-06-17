-- Revert ltree_del_col

BEGIN;

DROP FUNCTION goiardi.delete_search_collection(col text, m_organization_id int);

COMMIT;
