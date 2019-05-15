.. _persistence:

General Database Options
========================

There are two general options that can be set for either database: ``--db-pool-size`` and ``--max-connections`` (and their configuration file equivalents ``db-pool-size`` and ``max-connections``). ``--db-pool-size`` sets the number of idle connections to keep open to the database, and ``--max-connections`` sets the maximum number of connections to open on the database. If they are not set, the default behavior is to keep no idle connections alive and to have unlimited connections to the database.

It should go without saying that these options don't do much if you aren't using one of the SQL backends.

Of the two databases available, PostgreSQL is the better supported and recommended configuration. MySQL still works, of course, but it can't take advantage of some of the very helpful Postgres features.

MySQL mode
----------

Goiardi can use MySQL to store its data, instead of keeping all its data in memory (and optionally freezing its data to disk for persistence).

If you want to use MySQL, you (unsurprisingly) need a MySQL installation that goiardi can access. This document assumes that you are able to install, configure, and run MySQL.

Once the MySQL server is set up to your satisfaction, you'll need to install sqitch to deploy the schema, and any changes to the database schema that may come along later. It can be installed out of CPAN or homebrew; see "Installation" on http://sqitch.org for details.

The sqitch MySQL tutorial at https://metacpan.org/pod/sqitchtutorial-mysql explains how to deploy, verify, and revert changes to the database with sqitch, but the basic steps to deploy the schema are:

* Create goiardi's database: ``mysql -u root --execute 'CREATE DATABASE goiardi'``
* Optionally, create a separate mysql user for goiardi and give it permissions
  on that database.
* In sql-files/mysql-bundle, deploy the bundle: ``sqitch deploy db:mysql://root[:<password>]@/goiardi``

To update an existing database deployed by sqitch, run the ``sqitch deploy`` command above again.

If you really really don't want to install sqitch, apply each SQL patch in sql-files/mysql-bundle by hand in the same order they're listed in the sqitch.plan file.

The above values are for illustration, of course; nothing requires goiardi's database to be named "goiardi". Just make sure the right database is specified in the config file.

Set ``use-mysql = true`` in the configuration file, or specify ``--use-mysql`` on the command line. If both the ``-D``/``--data-file`` flag and ``--use-mysql`` are used at the same time, an error will be printed to the log and the data file option will be ignored.

An example configuration is available in ``etc/goiardi.conf-sample``, and is given below::

    [mysql]
        username = "foo" # technically optional, although you probably want it
        password = "s3kr1t" # optional, if you have no password set for MySQL
        protocol = "tcp" # optional, but set to "unix" for connecting to MySQL
                 # through a Unix socket.
        address = "localhost"
        port = "3306" # optional, defaults to 3306. Not used with sockets.
        dbname = "goiardi"
        # See https://github.com/go-sql-driver/mysql#parameters for an
        # explanation of available parameters
        [mysql.extra_params]
            tls = "false"

A similar example for configuring MySQL access via the command line is below:

``goiardi -A --conf-root=/Users/jeremy/etc/goiardi --ipaddress="0.0.0.0" --log-level="debug" --local-filestore-dir=/var/lib/goiardi/lfs --db-pool-size=25 --use-mysql --mysql-username=goiardi --mysql-address=localhost --mysql-extra-params=tls:false -i /var/goiardi/idx.bin --mysql-dbname=goiardi``

Postgres mode
-------------

Goiardi can also use Postgres as a backend for storing its data, instead of using MySQL or the in-memory data store. The overall procedure is pretty similar to setting up goiardi to use MySQL. Specifically for Postgres, you may want to create a database especially for goiardi, but it's not mandatory. If you do, you may also want to create a user for it. If you decide to do that:

* Create the user: ``$ createuser goiardi <additional options>``
* Create the database, if you decided to: ``$ createdb goiardi_db <additional options>``. If you created a user, make it the owner of the goiardi db with ``-O goiardi``.

After you've done that, or decided to use an existing database and user, deploy the sqitch bundle in sql-files/postgres-bundle. If you're using the default Postgres user on the local machine, ``sqitch deploy db:pg:<dbname>`` will be sufficient. Otherwise, the deploy command will be something like ``sqitch deploy db:pg://user:password@localhost/goiardi_db``.

The Postgres sqitch tutorial at https://metacpan.org/pod/sqitchtutorial explains more about how to use sqitch and Postgres.

Set ``use-postgresql`` in the configuration file, or specify ``--use-postgresql`` on the command line. Specifying both ``-D``/``--data-file`` flag and ``--use-postgresql`` at the same time will print an error to the log and ignore the data file setting, like how it works in MySQL mode. MySQL and Postgres cannot be used at the same time, also, and will result in a fatal error.

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
