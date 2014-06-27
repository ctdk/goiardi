-- Verify cookbook_insert_update

BEGIN;

SELECT goiardi.merge_cookbooks('moo');
SELECT updated_at INTO TEMPORARY old_cook FROM goiardi.cookbooks WHERE name = 'moo';
SELECT goiardi.merge_cookbooks('moo');
SELECT c.updated_at FROM goiardi.cookbooks c, old_cook WHERE name = 'moo' AND c.updated_at <> old_cook.updated_at;

ROLLBACK;
