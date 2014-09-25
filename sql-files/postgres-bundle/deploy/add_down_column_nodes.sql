-- Deploy add_down_column_nodes
-- requires: nodes

BEGIN;

ALTER TABLE goiardi.nodes ADD COLUMN is_down bool DEFAULT FALSE;
CREATE INDEX node_is_down ON goiardi.nodes(is_down);

COMMIT;
