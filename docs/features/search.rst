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

In testing, goiardi with postgres search can handle 10,000 nodes without any particular problem. Simple queries complete reasonably quickly, but more complex queries can take longer. In the most recent tests, on a 2014 MacBook Pro with 16GB of RAM and a totally untuned PostgreSQL installation, executing the search query equivalent to "data_center:Vagrantheim" directly into the database with 10,000 nodes consistently took about 40-60 milliseconds. The equivalent of "data_center:Vagrantheim AND name:server2*" took between 3 and 4 seconds, while "data_center:Vagrantheim AND name:(server2* OR server4*)" took about 7-8 seconds. It is expected that with proper tuning, and as this feature matures, these numbers will go down. It's also worth mentioning that when using knife search, the whole process takes considerably longer anyway.

The postgres search should be able to handle almost any query you throw at it, but it's definitely possible to craft a query that goiardi will fail to handle correctly. Particularly, if you're using fuzzy or distance searches, it will probably not return what you want. This postgres search should handle all normal cases, however.

The postgres based search still uses the same Solr syntax that chef search traditionally uses, but the Solr queries are parsed out and used to generate SQL queries for searching. There is likely room for improvement with the generated queries. An intriguing possibility for down the road is to allow an alternate query syntax that more closely reflects postgres' capabilities with these indexes.

The biggest issue between the standard Solr search with Chef and the goiardi Postgres based search is that ltree indices in Postgres can only use alphanumeric characters (plus _), with "." as a path separator. Since attributes can have whatever arbitrary characters you want in them, goiardi strips those characters out when they're indexed and when searching. This is not usually a problem, but could lead to strange results if you had something like "/dev/xvda1" AND "dev_xvda1" as attribute names in a node.

One difference that's worth mentioning is that you can start a search term with a wildcard character with the postgres search, unlike with the Solr searches.

To use the postgres search in your goiardi installation, you must:

a) be using postgres (duh) and
b) enable it in your goiardi.conf file with ``pg-search = true``.

It is strongly recommended that you also set ``convert-search = true``, because the postgres search uses the dot separator between path items instead of the underscore, and this will break existing search queries. If ``index-file`` is set, goiardi will print a warning that it's not very useful to have the index file enabled, but it's not a fatal error.

**NB:** It is also *very* strongly recommended, especially if you run chef-client frequently in a cron or as a daemon, that you periodically reindex the search_items table. Otherwise, the indexes can grow to ridiculous sizes and you'll be wondering why you're running out of space for no clear reason. The procedure is simple, however: add a command like ``echo 'REINDEX TABLE goiardi.search_items; VACUUM;' | /usr/bin/psql -d <GOIARDI_DB_NAME>`` to the crontab of your postgres user (probably postgres), or some other user account with rights to the goiardi database, and that will take care of the reindexing for you. Running this command daily is a good idea, but you can experiment with reindexing at different time frames and see what works best for you. The act of reindexing itself does not appear to be particularly stressful, but of course finding a relatively quiet time to do the reindexing is probably a good idea.

Also note that as this is a pretty feature the details are subject to change. In particular, the indexes on the search_items table are likely not to be optimal; you should experiment with tweaking those as you see fit, and if you find something (or the removal of something) that works especially well, please let me know.

This has been around for a while now and it's been tested pretty thoroughly and  been running reliably in production it may still have some problems. If so, `filing issues <https://github.com/ctdk/goiardi/issues>`_ is appreciated.

Search index trimming
---------------------

One option added in version 0.11.3 is the ability to trim the length of values (not keys) that will be stored in the index with ``-T/--index-val-trim``. This leads to smaller indexes and, hopefully, lower memory usage. Currently, it defaults to 0 (meaning that no values in the index will be trimmed), but this behavior will change with the next major release.

Some thought should be put in to what the trim length should be. If it's too short, searches may have unexpected problems. In testing with chef-pedant locally, trimming values down to 50 characters caused some search tests to break, while 100 characters worked fine. A good value generally is 100 characters, but you may need to adjust the trim value and test until you find a good number if 100 characters doesn't work well for you.

Rebuilding search indexes
-------------------------

Newer versions of ``knife`` have removed the ``knife index rebuild`` command. While Chef Server hasn't needed it for a long time, goiardi still makes use of that functionality, so a < LINK TO PLUGIN > has been made to reintroduce that functionality.
