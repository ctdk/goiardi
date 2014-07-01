-- Deploy client_rename
-- requires: clients
-- requires: goiardi_schema

BEGIN;

CREATE OR REPLACE FUNCTION goiardi.rename_client(old_name text, new_name text) RETURNS VOID AS
$$
DECLARE
	u_id bigint;
	u_name text;
BEGIN
	SELECT id, name INTO u_id, u_name FROM goiardi.users WHERE name = new_name;
	IF FOUND THEN
		RAISE EXCEPTION 'a user with id % named % was found that would conflict with this client', u_id, u_name;
	END IF;
	BEGIN
		UPDATE goiardi.clients SET name = new_name WHERE name = old_name;
	EXCEPTION WHEN unique_violation THEN
		RAISE EXCEPTION 'Client % already exists, cannot rename %', old_name, new_name;
	END;
END;
$$
LANGUAGE plpgsql;

COMMIT;
