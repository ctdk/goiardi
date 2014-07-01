-- Deploy data_bag_items
-- requires: data_bags
-- requires: goiardi_schema

BEGIN;

CREATE TABLE goiardi.data_bag_items (
	id bigserial,
	name text not null,
	orig_name text not null,
	data_bag_id bigint not null,
	raw_data bytea,
	created_at timestamp with time zone not null,
	updated_at timestamp with time zone not null,
	primary key(id),
	FOREIGN KEY(data_bag_id)
		REFERENCES goiardi.data_bags(id)
		ON DELETE RESTRICT,
	unique(data_bag_id, name),
	unique(data_bag_id, orig_name)
);
ALTER TABLE goiardi.data_bag_items ALTER raw_data SET STORAGE EXTERNAL;

COMMIT;
