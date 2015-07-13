.. _search:

Search
======

Goiardi currently has two different ways of running searches: the original and default ersatz Solr implementation, and a Postgres based search using ltree and trigrams with some new database tables. Both use the usual Solr syntax that chef expects, but are quite different under the hood.

Additional different search backends are now a possibility as well; goiardi search's archictecture has changed to make it easier to add new search backends, like actual Solr search or what have you.

Ersatz Solr Search
------------------

Nothing special needs to be done to use this search. It remains the default search implementation, and the only choice for the in-memory/file based storage and MySQL. It works well for smaller installations, but when you get in the neighborhood of hundreds of nodes it begins to get bogged down.

Postgres Search
---------------

Starting with goiardi version 0.10.0, there is an optional PostgreSQL based search. It uses the same solr parser that the default search backend uses, but instead of using tries to search for objects, it uses ltree and trigrams to search for values stored in a separate table. The postgres search is able to use the same solr query parser the original search uses to create postgres queries from the solr queries.

In testing, goiardi with postgres search can handle 10,000 nodes without any problem. Simple queries complete very quickly, but more complex queries can take longer. TODO: more benchmarks

The postgres search should be able to handle almost any query you throw at it, but it's definitely possible to craft a query that goiardi will fail to handle correctly. Particularly, if you're using fuzzy or distance searches, it will probably not return what you want. This postgres search should handle all normal cases, however.

The postgres based search still uses the same Solr syntax that chef search traditionally uses, but the Solr queries are parsed out and used to generate SQL queries for searching. There is likely room for improvement with the generated queries. An intriguing possibility for down the road is to allow an alternate query syntax that more closely reflects postgres' capabilities with these indexes.

The biggest issue between the standard Solr search with Chef and the goiardi Postgres based search is that ltree indices in Postgres can only use alphanumeric characters (plus _), with "." as a path separator. Since attributes can have whatever arbitrary characters you want in them, goiardi strips those characters out when they're indexed and when searching. This is not usually a problem, but could lead to strange results if you had something like "/dev/xvda1" AND "dev_xvda1" as attribute names in a node.

One difference that's worth mentioning is that you can start a search term with a wildcard character with the postgres search, unlike with the Solr searches.

To use the postgres search in your goiardi installation, you must:

a) be using postgres (duh) and
b) enable it in your goiardi.conf file with `pg-search = true`.

It is strongly recommended that you also set `convert-search = true`, because the postgres search uses the dot separator between path items instead of the underscore, and this will break existing search queries. If `index-file` is set, goiardi will print a warning that it's not very useful to have the index file enabled, but it's not a fatal error.

This is very new, and while it's been tested pretty thoroughly and has been running reliably in production for a while it may still have some problems. If so, [filing issues](https://github.com/ctdk/goiardi/issues) is appreciated.
