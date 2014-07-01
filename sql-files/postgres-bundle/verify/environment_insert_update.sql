-- Verify environment_insert_update

BEGIN;

SELECT goiardi.merge_environments('moo', 'moo desc', NULL, NULL, NULL);
SELECT id FROM goiardi.environments WHERE name = 'moo' AND description = 'moo desc';
SELECT goiardi.merge_environments('moo', 'moohoo', NULL, NULL, NULL);
SELECT id FROM goiardi.environments WHERE name = 'moo' AND description = 'moohoo'; 

ROLLBACK;
