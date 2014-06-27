-- Verify client_rename

BEGIN;

INSERT INTO goiardi.clients (name, created_at, updated_at) VALUES ('foobar', NOW(), NOW());
SELECT id FROM goiardi.clients WHERE name = 'foobar';
SELECT goiardi.rename_client('foobar', 'foobaz');
SELECT id FROM goiardi.clients WHERE name = 'foobaz';

ROLLBACK;
