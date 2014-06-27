-- Verify user_rename

BEGIN;

INSERT INTO goiardi.users (name, created_at, updated_at) VALUES ('foobar', NOW(), NOW());
SELECT id FROM goiardi.users WHERE name = 'foobar';
SELECT goiardi.rename_user('foobar', 'foobaz', 1);
SELECT id FROM goiardi.users WHERE name = 'foobaz';

ROLLBACK;
