-- Deploy log_infos
-- requires: clients
-- requires: users
-- requires: goiardi_schema

BEGIN;

CREATE TYPE goiardi.log_action AS ENUM ( 'create', 'delete', 'modify');
CREATE TYPE goiardi.log_actor AS ENUM ( 'user', 'client');
CREATE TABLE goiardi.log_infos (
	id bigserial,
	actor_id bigint not null default 0,
	actor_info text,
	actor_type goiardi.log_actor NOT NULL,
	organization_id bigint not null default '1',
	time timestamp with time zone default current_timestamp,
	action goiardi.log_action not null,
	object_type text not null,
	object_name text not null,
	extended_info text,
	primary key(id)
);

CREATE INDEX log_infos_actor ON goiardi.log_infos(actor_id);
CREATE INDEX log_infos_action ON goiardi.log_infos(action);
CREATE INDEX log_infos_obj ON goiardi.log_infos(object_type, object_name);
CREATE INDEX log_infos_time ON goiardi.log_infos(time);
CREATE INDEX log_info_orgs ON goiardi.log_infos(organization_id);
ALTER TABLE goiardi.log_infos ALTER extended_info SET STORAGE EXTERNAL;

COMMIT;
