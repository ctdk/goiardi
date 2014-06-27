-- Deploy reports

BEGIN;

CREATE TYPE goiardi.report_status AS ENUM ( 'started', 'success', 'failure' );

CREATE TABLE goiardi.reports (
	id bigserial,
	run_id uuid not null,
	node_name varchar(255),
	organization_id bigint not null default 1,
	start_time timestamp with time zone,
	end_time timestamp with time zone,
	total_res_count int default 0,
	status goiardi.report_status,
	run_list text,
	resources bytea,
	data bytea,
	created_at timestamp with time zone not null,
	updated_at timestamp with time zone not null,
	primary key(id),
	unique(run_id)
);

CREATE INDEX report_organization_id ON goiardi.reports(organization_id);
CREATE INDEX report_node_organization ON goiardi.reports(node_name, organization_id);
ALTER TABLE goiardi.reports ALTER run_list SET STORAGE EXTERNAL;
ALTER TABLE goiardi.reports ALTER resources SET STORAGE EXTERNAL;
ALTER TABLE goiardi.reports ALTER data SET STORAGE EXTERNAL;

COMMIT;
