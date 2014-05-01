-- Deploy clients

BEGIN;

CREATE TABLE clients (
	id int not null auto_increment,
	name varchar(2048) not null,
	nodename varchar(2048),
	validator tinyint(4) default 0,
	admin tinyint(4) default 0,
	organization_id int not null default 1,
	public_key text,
	certificate text,
	created_at datetime not null,
	updated_at datetime not null,
	primary key(id),
	unique key(organization_id, name(250))
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

COMMIT;
