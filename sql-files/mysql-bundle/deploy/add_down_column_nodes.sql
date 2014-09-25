-- Deploy add_down_column_nodes
-- requires: nodes

BEGIN;

ALTER TABLE nodes ADD COLUMN is_down tinyint default 0, ADD INDEX(is_down);

COMMIT;
