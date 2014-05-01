-- Deploy sandboxes

BEGIN;

CREATE TABLE sandboxes (
	id int not null auto_increment,
	sbox_id varchar(32) not null,
	creation_time datetime not null,
	checksums blob,
	completed tinyint(4) default 0,
	primary key(id),
	unique key(sbox_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

COMMIT;
