-- Deploy node_statuses
-- requires: nodes

BEGIN;

CREATE TABLE node_statuses (
	id int not null auto_increment,
	node_id int not null,
	status enum('new', 'up', 'down') not null default 'new',
	updated_at datetime not null,
	PRIMARY KEY(id),
	INDEX(status),
	INDEX(updated_at),
	FOREIGN KEY(node_id)
		REFERENCES nodes(id)
		ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

COMMIT;
