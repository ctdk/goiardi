-- Deploy node_statuses
-- requires: nodes

BEGIN;

CREATE TYPE goiardi.status_node AS ENUM ( 'new', 'up', 'down' );
CREATE TABLE goiardi.node_statuses (
	id bigserial,
	node_id bigint not null,
	status goiardi.status_node not null default 'new',
	updated_at timestamp with time zone not null,
	PRIMARY KEY(id),
	FOREIGN KEY(node_id)
		REFERENCES goiardi.nodes(id)
		ON DELETE CASCADE
);
CREATE INDEX node_status_status ON goiardi.node_statuses(status);
CREATE INDEX node_status_time ON goiardi.node_statuses(updated_at);

COMMIT;
