-- Deploy cookbook_versions

BEGIN;

CREATE TABLE cookbook_versions (
	id int not null auto_increment,
	cookbook_id int not null,
	major_ver bigint not null,
	minor_ver bigint not null,
	patch_ver bigint not null default 0, -- the first two *must* be set,
					     -- the third not necessarily. This
					     -- may be better as a nullable
					     -- column though.
	frozen tinyint default 0,
	metadata blob,
	definitions blob,
	libraries blob,
	attributes blob,
	recipes blob,
	providers blob,
	resources blob,
	templates blob,
	root_files blob,
	files blob,
	created_at datetime not null,
	updated_at datetime not null,
	PRIMARY KEY(id),
	UNIQUE KEY(cookbook_id, major_ver, minor_ver, patch_ver),
	FOREIGN KEY (cookbook_id)
		REFERENCES cookbooks(id)
		ON DELETE RESTRICT,
	INDEX(frozen)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 ROW_FORMAT=COMPRESSED;

COMMIT;
