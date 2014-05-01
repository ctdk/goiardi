-- Deploy file_checksums

BEGIN;

CREATE TABLE file_checksums (
	id int not null auto_increment,
	org_id int not null default 0,
	checksum varchar(32),
	primary key(id),
	unique key(org_id, checksum)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

COMMIT;
