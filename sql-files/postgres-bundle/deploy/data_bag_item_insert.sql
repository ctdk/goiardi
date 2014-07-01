-- Deploy data_bag_item_insert
-- requires: data_bag_items
-- requires: data_bags
-- requires: goiardi_schema

BEGIN;

CREATE OR REPLACE FUNCTION goiardi.insert_dbi(m_data_bag_name text, m_name text, m_orig_name text, m_dbag_id bigint, m_raw_data bytea) RETURNS BIGINT AS
$$
DECLARE
	u BIGINT;
	dbi_id BIGINT;
BEGIN
	SELECT id INTO u FROM goiardi.data_bags WHERE id = m_dbag_id;
	IF NOT FOUND THEN
		RAISE EXCEPTION 'aiiiie! The data bag % was deleted from the db while we were doing something else', m_data_bag_name;
	END IF;

	INSERT INTO goiardi.data_bag_items (name, orig_name, data_bag_id, raw_data, created_at, updated_at) VALUES (m_name, m_orig_name, m_dbag_id, m_raw_data, NOW(), NOW()) RETURNING id INTO dbi_id;
	RETURN dbi_id;
END;
$$
LANGUAGE plpgsql;

COMMIT;
