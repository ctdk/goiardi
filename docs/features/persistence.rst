.. _persistence:

Database Options
========================

There are two general options that can be set for either database: ``--db-pool-size`` and ``--max-connections`` (and their configuration file equivalents ``db-pool-size`` and ``max-connections``). ``--db-pool-size`` sets the number of idle connections to keep open to the database, and ``--max-connections`` sets the maximum number of connections to open on the database. If they are not set, the default behavior is to keep no idle connections alive and to have unlimited connections to the database.

While MySQL was supported by goiardi previously, as of version 1.0.0 support for MySQL has been removed. Setting ``use-mysql`` will result in a fatal error. The only option for a database backend is Postgres at this point.

Postgres mode
-------------

Goiardi can use Postgres as a backend for storing its data, instead of using the in-memory data store. You may want to create a database especially for goiardi, but it's not mandatory. If you do, you may also want to create a user for it. If you decide to do that:

* Create the user: ``$ createuser goiardi <additional options>``
* Create the database, if you decided to: ``$ createdb goiardi_db <additional options>``. If you created a user, make it the owner of the goiardi db with ``-O goiardi``.

After you've done that, or decided to use an existing database and user, deploy the sqitch bundle in sql-files/postgres-bundle. If you're using the default Postgres user on the local machine, ``sqitch deploy db:pg:<dbname>`` will be sufficient. Otherwise, the deploy command will be something like ``sqitch deploy db:pg://user:password@localhost/goiardi_db``.

The Postgres sqitch tutorial at https://metacpan.org/pod/sqitchtutorial explains more about how to use sqitch and Postgres.

If you really really don't want to install sqitch, apply each SQL patch in sql-files/postgres-bundle by hand in the same order they're listed in the sqitch.plan file.

Set ``use-postgresql`` in the configuration file, or specify ``--use-postgresql`` on the command line. Specifying both ``-D``/``--data-file`` flag and ``--use-postgresql`` at the same time will print an error to the log and ignore the data file setting.

There is also an example Postgres configuration in the config file, and can be seen below::

    # PostgreSQL options. If "use-postgres" is set to true on the command line or in
    # the configuration file, connect to postgres with the options in [postgres].
    # These options are all strings. See
    # http://godoc.org/github.com/lib/pq#hdr-Connection_String_Parameters for details
    # on the connection parameters. All of these parameters are technically optional,
    # although chances are pretty good that you'd want to set at least some of them.
    [postgresql]
        username = "foo"
        password = "s3kr1t"
        host = "localhost"
        port = "5432"
        dbname = "mydb"
        sslmode = "disable"

A command line flag sample would be something like this:

``goiardi -A --conf-root=/etc/goiardi --ipaddress="0.0.0.0" --log-level="debug" --local-filestore-dir=/var/lib/goiardi/lfs --pg-search --convert-search --db-pool-size=25 --use-postgresql --postgresql-username=goiardi --postgresql-host=localhost --postgresql-dbname=goiardidb --postgresql-ssl-mode=disable``

Note regarding goiardi persistence and freezing data
----------------------------------------------------

As mentioned above, goiardi can now freeze its in-memory data store and index to disk if specified. It will save before quitting if the program receives a SIGTERM or SIGINT signal, along with saving every "freeze-interval" seconds automatically if there have been any changes.

Saving automatically helps guard against the case where the server receives a signal that it can't handle and forces it to quit. In addition, goiardi will not replace the old save files until the new one is all finished writing. However, it's still not anywhere near a real database with transaction protection, etc., so while it should work fine in the general case, possibilities for data loss and corruption do exist. The appropriate caution is warranted.

Goiardi's in-memory mode, with or without freezing data for persistence, is quite useful for testing, dev work, and the like, but is not recommended for long-term usage. In that situation, using Postgres is strongly recommended.
