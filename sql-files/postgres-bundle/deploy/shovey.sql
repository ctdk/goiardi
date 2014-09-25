-- Deploy shovey

BEGIN;

CREATE TYPE goiardi.shovey_output AS ENUM ( 'stdout', 'stderr' );

CREATE TABLE goiardi.shoveys (
	id bigserial,
	run_id uuid not null,
	command text,
	status text,
	timeout bigint default 300,
	quorum varchar(25) default '100%',
	created_at timestamp with time zone not null,
	updated_at timestamp with time zone not null,
	organization_id bigint not null default 1,
	primary key(id),
	unique(run_id)
);

CREATE TABLE goiardi.shovey_runs (
	id bigserial,
	shovey_uuid uuid not null,
	shovey_id bigint not null,
	node_name text,
	status text,
	ack_time timestamp with time zone,
	end_time timestamp with time zone,
	error text,
	exit_status smallint,
	primary key(id),
	unique(shovey_id, node_name),
	FOREIGN KEY (shovey_id)
		REFERENCES goiardi.shoveys(id)
		ON DELETE RESTRICT
);

CREATE TABLE goiardi.shovey_run_streams (
	id bigserial,
	shovey_run_id bigint not null,
	seq int not null,
	output_type goiardi.shovey_output,
	output text,
	is_last bool,
	created_at timestamp with time zone not null,
	primary key (id),
	unique(shovey_run_id, output_type, seq),
	FOREIGN KEY (shovey_run_id)
		REFERENCES goiardi.shovey_runs(id)
		ON DELETE RESTRICT
);

CREATE INDEX shoveys_status ON goiardi.shoveys(status);
CREATE INDEX shovey_organization_id ON goiardi.shoveys(organization_id);
CREATE INDEX shovey_organization_run_id ON goiardi.shoveys(run_id, organization_id);
CREATE INDEX shovey_run_run_id ON goiardi.shovey_runs(shovey_uuid);
CREATE INDEX shovey_run_node_name ON goiardi.shovey_runs(node_name);
CREATE INDEX shovey_run_status ON goiardi.shovey_runs(status);
CREATE INDEX shovey_uuid_node ON goiardi.shovey_runs(shovey_uuid, node_name);
CREATE INDEX shovey_stream ON goiardi.shovey_run_streams(shovey_run_id, output_type);

COMMIT;
