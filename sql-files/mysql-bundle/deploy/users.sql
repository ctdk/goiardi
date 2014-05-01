-- Deploy users

BEGIN;

CREATE TABLE users (
	id int not null auto_increment,
	name varchar(255) not null,
	displayname varchar(1024),
	email varchar(255),
	admin tinyint(4) default 0,
	public_key text,
	passwd varchar(128),
	salt varbinary(64),
	created_at datetime not null,
	updated_at datetime not null,
	primary key(id),
	unique key(name),
	unique key(email)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

COMMIT;
