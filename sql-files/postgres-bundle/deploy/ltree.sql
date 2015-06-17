-- Deploy ltree

BEGIN;
CREATE EXTENSION ltree SCHEMA goiardi;
CREATE EXTENSION pg_trgm SCHEMA goiardi;

CREATE TABLE goiardi.search_collections (
	id bigserial,
	organization_id bigint not null default 1,
	name text,
	PRIMARY KEY(id),
	UNIQUE(organization_id, name)
);

CREATE TABLE goiardi.search_items (
	id bigserial,
	organization_id bigint not null default 1,
	search_collection_id bigint not null,
	item_name text,
	value text,
	path goiardi.ltree,
	PRIMARY KEY(id),
	FOREIGN KEY (search_collection_id)
		REFERENCES goiardi.search_collections(id)
		ON DELETE RESTRICT
);


CREATE INDEX search_col_name ON goiardi.search_collections(name);
CREATE INDEX search_org_id ON goiardi.search_items(organization_id);
CREATE INDEX search_org_col ON goiardi.search_items(organization_id, search_collection_id);
CREATE INDEX search_gist_idx ON goiardi.search_items USING gist (path);
CREATE INDEX search_btree_idx ON goiardi.search_items USING btree(path);
CREATE INDEX search_org_col_name ON goiardi.search_items(organization_id, search_collection_id, item_name);
CREATE INDEX search_item_val_trgm ON goiardi.search_items USING gist (value goiardi.gist_trgm_ops);
CREATE INDEX search_multi_gist_idx ON goiardi.search_items USING gist (path, value goiardi.gist_trgm_ops);
CREATE INDEX search_val ON goiardi.search_items(value);

COMMIT;
