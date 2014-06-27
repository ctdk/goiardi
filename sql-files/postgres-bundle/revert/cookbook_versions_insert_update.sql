-- Revert cookbook_versions_insert_update

BEGIN;

DROP FUNCTION goiardi.merge_cookbook_versions(c_id bigint, is_frozen bool, defb bytea, libb bytea, attb bytea, recb bytea, prob bytea, resb bytea, temb bytea, roob bytea, filb bytea, metb bytea, maj bigint, min bigint, patch bigint);

COMMIT;
