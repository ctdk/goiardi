.. _search:

Search
======

Goiardi currently has two different ways of running searches: the original and default ersatz Solr implementation, and a Postgres based search using ltree and trigrams with some new database tables. Both use the usual Solr syntax that chef expects, but are quite different under the hood.

Additional different search backends are now a possibility as well; the archictecture of the backend has changed to make it easier to add new search backends, like actual Solr search or what have you.

Ersatz Solr Search
------------------

Nothing special needs to be done to use this search. It remains the default search implementation, and the only choice for the in-memory/file based storage and MySQL. It works well, but when you get in the neighborhood of hundreds of nodes it begins to get bogged down.

Postgres Search
---------------

Starting with goiardi version 0.10.0, there is an optional PostgreSQL based search. It uses the same solr parser that the default search backend uses, but instead of using tries to search for objects, it uses ltree and trigrams to search for values stored in a separate table. The postgres search is able to use the same solr query parser the original search uses to create postgres queries from the solr queries.

In testing, goiardi with postgres search can handle 10,000 nodes without any problem. Simple queries complete very quickly, but more complex queries can take longer. TODO: more benchmarks

The postgres search should be able to handle almost any query you throw at it, but it's definitely possible to craft a query that goiardi will fail to handle correctly. Particularly, if you're using fuzzy or distance searches, it will probably not return what you want. This postgres search should handle all normal cases, however.
