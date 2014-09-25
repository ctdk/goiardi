-- Deploy node_latest_statuses
-- requires: node_statuses

BEGIN;

CREATE OR REPLACE VIEW goiardi.node_latest_statuses(
	id,
	name,
	chef_environment,
	run_list,
	automatic_attr,
	normal_attr,
	default_attr,
	override_attr,
	is_down,
	status,
	updated_at)
AS
SELECT DISTINCT ON (n.id)
	n.id,
	n.name,
	n.chef_environment,
	n.run_list,
	n.automatic_attr,
	n.normal_attr,
	n.default_attr,
	n.override_attr,
	n.is_down,
	ns.status,
	ns.updated_at
	FROM goiardi.nodes n INNER JOIN goiardi.node_statuses ns ON n.id = ns.node_id
	ORDER BY n.id, ns.updated_at DESC;

COMMIT;
