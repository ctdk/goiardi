-- Deploy node_search

BEGIN;

CREATE TABLE node_search
(
  name text NOT NULL,
  field text NOT NULL,
  value text,
  CONSTRAINT node_search_pk PRIMARY KEY (name, field)
)
WITH (
  OIDS=FALSE
);
ALTER TABLE node_search
  OWNER TO goiardi;

CREATE INDEX value_fulltext
  ON node_search
  USING gin
  (to_tsvector('english'::regconfig, value));

COMMIT;
