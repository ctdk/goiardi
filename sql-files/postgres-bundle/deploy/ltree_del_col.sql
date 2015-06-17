-- Deploy ltree_del_col

BEGIN;

CREATE OR REPLACE FUNCTION goiardi.delete_search_collection(col text, m_organization_id int) RETURNS VOID AS
$$
DECLARE
	sc_id bigint;
BEGIN
	SELECT id INTO sc_id FROM goiardi.search_collections WHERE name = col AND organization_id = m_organization_id;
	IF NOT FOUND THEN
		RAISE EXCEPTION 'The collection % does not exist!', col;
	END IF;
	DELETE FROM goiardi.search_items WHERE organization_id = m_organization_id AND search_collection_id = sc_id;
	DELETE FROM goiardi.search_collections WHERE organization_id = m_organization_id AND id = sc_id;
END;
$$
LANGUAGE plpgsql;

COMMIT;
