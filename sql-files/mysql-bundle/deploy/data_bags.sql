-- Deploy data_bags

BEGIN;

CREATE TABLE data_bags (
	id int not null auto_increment,
	name varchar(255) not null,
	created_at datetime not null,
	updated_at datetime not null,
	primary key(id),
	unique key(name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

COMMIT;
