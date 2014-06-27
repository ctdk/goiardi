-- Deploy user_rename
-- requires: users
-- requires: goiardi_schema

BEGIN;

CREATE OR REPLACE FUNCTION goiardi.rename_user(old_name text, new_name text, m_organization_id int) RETURNS VOID AS
$$
DECLARE
	c_id bigint;
	c_name text;
BEGIN
	SELECT id, name INTO c_id, c_name FROM goiardi.clients WHERE name = new_name AND organization_id = m_organization_id;
	IF FOUND THEN
		RAISE EXCEPTION 'a client with id % named % was found that would conflict with this user', c_id, c_name;
	END IF;
	BEGIN
		UPDATE goiardi.users SET name = new_name WHERE name = old_name;
	EXCEPTION WHEN unique_violation THEN
		RAISE EXCEPTION 'User % already exists, cannot rename %', old_name, new_name;
	END;
END;
$$
LANGUAGE plpgsql;

COMMIT;
