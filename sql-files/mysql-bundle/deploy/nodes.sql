-- Deploy nodes

BEGIN;

CREATE TABLE nodes (
	id int not null auto_increment,
	name varchar(255) not null,
	chef_environment varchar(255) not null default "_default", 
	run_list blob,
	automatic_attr blob,
	normal_attr blob,
	default_attr blob,
	override_attr blob,
	created_at datetime not null,
	updated_at datetime not null,
	primary key(id),
	unique key(name),
	key(chef_environment)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 ROW_FORMAT=COMPRESSED;

COMMIT;
