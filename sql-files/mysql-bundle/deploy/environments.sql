-- Deploy environments

BEGIN;

CREATE TABLE environments (
	id int not null auto_increment,
	name varchar(255) not null,
	description text,
	default_attr blob,
	override_attr blob,
	cookbook_vers blob, -- make a blob for now, may bust out to a table
	created_at datetime not null,
	updated_at datetime not null,
	PRIMARY KEY(id),
	UNIQUE KEY(name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 ROW_FORMAT=COMPRESSED;
INSERT INTO environments (id, name, description, created_at, updated_at) VALUES (1, "_default", "The default Chef environment", NOW(), NOW());

COMMIT;
