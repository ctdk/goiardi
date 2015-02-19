-- Verify node_search

BEGIN;

SELECT name FROM node_search where value @@ 'asd:*'::tsquery;

ROLLBACK;
