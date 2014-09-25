-- Deploy node_latest_statuses

BEGIN;

CREATE OR REPLACE VIEW node_latest_statuses 
	(id,
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
	SELECT DISTINCT n.id, 
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
	FROM nodes n 
	INNER JOIN node_statuses ns ON n.id = ns.node_id 
	WHERE ns.id IN 
	(select max(id) from node_statuses GROUP BY node_id) 
	ORDER BY n.id;

COMMIT;
