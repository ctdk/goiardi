-- Deploy shovey

BEGIN;

CREATE TABLE shoveys (
	id int not null auto_increment,
	run_id varchar(36) not null,
	command text,
	status varchar(30),
	timeout int default 300,
	quorum varchar(25) default "100%",
	created_at datetime not null,
	updated_at datetime not null,
	organization_id int not null default 1,
	primary key(id),
	unique key(run_id),
	index(organization_id),
	index(run_id, organization_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

CREATE TABLE shovey_runs (
	id int not null auto_increment,
	shovey_uuid varchar(36) not null,
	shovey_id int not null,
	node_name varchar(255) not null,
	status varchar(30),
	ack_time datetime,
	end_time datetime,
	error text,
	exit_status tinyint unsigned,
	PRIMARY KEY(id),
	UNIQUE KEY(shovey_id, node_name),
	FOREIGN KEY (shovey_id)
		REFERENCES shoveys(id)
		ON DELETE RESTRICT,
	INDEX(shovey_uuid),
	INDEX(node_name),
	INDEX(status),
	INDEX(shovey_uuid, node_name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 ROW_FORMAT=COMPRESSED;

CREATE TABLE shovey_run_streams (
	id int not null auto_increment,
	shovey_run_id int not null,
	seq int not null,
	output_type enum('stdout', 'stderr'),
	output mediumtext,
	is_last tinyint default 0,
	created_at datetime not null,
	PRIMARY KEY(id),
	UNIQUE KEY(shovey_run_id, output_type, seq),
	FOREIGN KEY (shovey_run_id)
		REFERENCES shovey_runs(id)
		ON DELETE RESTRICT,
	INDEX(shovey_run_id, output_type)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 ROW_FORMAT=COMPRESSED;

COMMIT;
