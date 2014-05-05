Goiardi
=======

Goiardi is an implementation of the Chef server (http://www.opscode.com) written
in Go. It can either run entirely in memory with the option to save and load the
in-memory data and search indexes to and from disk, drawing inspiration from 
chef-zero, or it can use MySQL as its storage backend.

It is a work in progress. At the moment normal functionality as tested with 
knife works, and chef-client runs complete successfully. At this point, almost
all chef-pendant tests successfully successfully run, with a few disagreements
that don't impact the clients. It does pretty well against the official 
chef-pedant, but because goiardi handles some authentication matters a little 
differently than the official chef-server, there is also a fork of chef-pedant 
located at https://github.com/ctdk/chef-pedant that's more custom tailored to
goiardi.

Many go tests are present as well in different goiardi subdirectories.

DEPENDENCIES
------------

Goiardi currently has five dependencies: go-flags, go-cache, go-trie, toml, and
the mysql driver from go-sql-driver.

To install them, run:

```
   go get github.com/jessevdk/go-flags
   go get github.com/pmylund/go-cache
   go get github.com/ctdk/go-trie/gtrie
   go get github.com/BurntSushi/toml
   go get github.com/go-sql-driver/mysql
```

from your $GOROOT.

If you would like to modify the search grammar, you'll need the `peg` package.
To install that, run

```
   go get github.com/pointlander/peg
```

In the `search/` directory, run `peg -switch -inline search-parse.peg` to
generate the new grammar. If you don't plan on editing the search grammar,
though, you won't need that.

INSTALLATION
------------

1. Install go. (http://golang.org/doc/install.html) You may need to upgrade to
   go 1.2 to compile all the dependencies.

2. Make sure your $GOROOT and PATH are set up correctly per the Go installation
   instructions.

3. Download goairdi

```
   go get github.com/ctdk/goiardi
```

4. Run tests, if desired. Several goiardi subdirectories have go tests, and
   chef-pedant can and should be used for testing goiardi as well.

5. Install the goiardi binaries.

```
   go install github.com/ctdk/goiardi
```

6. Run goiardi.

```
   goiardi <options>
```

   You can get a list of command-line options with the '-h' flag. 

   Goiardi can also take a config file, run like `goiardi -c 
   /path/to/conf-file`. See `etc/goiardi.conf-sample` for an example documented
   configuration file. Options in the configuration file share the same name
   as the long command line arguments (so, for example, `--ipaddress=127.0.0.1`
   on the command line would be `ipaddress = "127.0.0.1"` in the config file.

   Currently available command line and config file options:

```
   -v, --version          Print version info.
   -V, --verbose          Show verbose debug information. Repeat for more
			  verbosity.
   -c, --config=          Specify a config file to use.
   -I, --ipaddress=       Listen on a specific IP address.
   -H, --hostname=        Hostname to use for this server. Defaults to hostname
                          reported by the kernel.
   -P, --port=            Port to listen on. If port is set to 443, SSL will be
                          activated. (default: 4545)
   -i, --index-file=      File to save search index data to.
   -D, --data-file=       File to save data store data to.
   -F, --freeze-interval= Interval in seconds to freeze in-memory data
                          structures to disk (requires -i/--index-file and
                          -D/--data-file options to be set). (Default 300
                          seconds/5 minutes.)
   -L, --log-file=        Log to file X
       --time-slew=       Time difference allowed between the server's clock at
                          the time in the X-OPS-TIMESTAMP header. Formatted like
                          5m, 150s, etc. Defaults to 15m.
       --conf-root=       Root directory for configs and certificates. Default:
                          the directory the config file is in, or the current
                          directory if no config file is set.
   -A, --use-auth         Use authentication. Default: false.
       --use-ssl          Use SSL for connections. If --port is set to 433, this
                          will automatically be turned on. If it is set to 80,
                          it will automatically be turned off. Default: off.
                          Requires --ssl-cert and --ssl-key.
       --ssl-cert=        SSL certificate file. If a relative path, will be set
                          relative to --conf-root.
       --ssl-key=         SSL key file. If a relative path, will be set relative
                          to --conf-root.
       --https-urls       Use 'https://' in URLs to server resources if goiardi
                          is not using SSL for its connections. Useful when
                          goiardi is sitting behind a reverse proxy that uses
                          SSL, but is communicating with the proxy over HTTP.
       --disable-webui    If enabled, disables connections and logins to goiardi
                          over the webui interface.
       --use-mysql        Use a MySQL database for data storage. Configure
                          database options in the config file.
       --local-filestore-dir= Directory to save uploaded files in. Optional when
                          running in in-memory mode, *mandatory* for SQL
                          mode.
       --log-events       Log changes to chef objects.
```

   Options specified on the command line override options in the config file.

For more documentation on Chef, see (http://docs.opscode.com).

If goiardi is not running in use-auth mode, it does not actually care about .pem
files at all. You still need to have one to keep knife and chef-client happy. 
It's like chef-zero in that regard.

If goiardi is running in use-auth mode, then proper keys are needed. When 
goiardi is started, if the chef-webui and chef-validator clients, and the admin 
user, are not present, it will create new keys in the --conf-root directory. Use
them as you would normally for validating clients, performing tasks with the
admin user, or using chef-webui if webui will run in front of goiardi.

*Note:* The admin user, when created on startup, does not have a password. This
prevents logging in to the webui with the admin user, so a password will have to
be set for admin before doing so.

### MySQL mode

Goiardi can now use MySQL to store its data, instead of keeping all its data 
in memory (and optionally freezing its data to disk for persistence).

If you want to use MySQL, you (unsurprisingly) need a MySQL installation that
goiardi can access. This document assumes that you are able to install, 
configure, and run MySQL.

Once the MySQL server is set up to your satisfaction, you'll need to install
sqitch to deploy the schema, and any changes to the database schema that may come
along later. It can be installed out of CPAN or homebrew; see "Installation" on
http://sqitch.org for details.

The sqitch MySQL tutorial at https://metacpan.org/pod/sqitchtutorial-mysql
explains how to deploy, verify, and revert changes to the database with sqitch,
but the basic steps to deploy the schema are:

* Create goiardi's database: `mysql -u root --execute 'CREATE DATABASE goiardi'`
* Optionally, create a separate mysql user for goiardi and give it permissions
  on that database.
* In sql-files/mysql-bundle, deploy the bundle: `sqitch deploy db:mysql://root@<password>/goiardi`

If you really really don't want to install sqitch, apply each SQL patch in
sql-files/mysql-bundle by hand in the same order they're listed in the
sqitch.plan file.

The above values are for illustration, of course; nothing requires goiardi's
database to be named "goiardi". Just make sure the right database is specified 
in the config file.

Set `use-mysql = true` in the configuration file, or specify `--use-mysql` on
the command line. It is an error to specify both the `-D`/`--data-file` flag and
`--use-mysql` at the same time.

At this time, the mysql connection options have to be defined in the config
file. An example configuration is available in `etc/goiardi.conf-sample`, and is
given below:

```
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
```

### Event Logging

Goiardi has optional event logging. When enabled with the `--log-events` command
line option, or with the `"log-events"` option in the config file, changes to
clients, users, cookbooks, data bags, environments, nodes, and roles will be
tracked. The event log can be viewed through the /events API endpoint.

The event API endpoints work as follows:

> `GET /events` - optionally taking `offset` and `limit` query parameters.
>
> List the logged events, starting with the most recent. Use the `offset` and
> `limit` query parameters to view smaller chunks of the event log at one time.

> `DELETE /events?purge=1234` - purge logged events older than the given id from
> the event log.

> `GET /events/1234` - get a single logged event with the given id.

> `DELETE /events/1234` - delete a single logged event from the event log.

A user or client must be an administrator account to use the `/events` endpoint.

The data returned from the event log should look something like this:

```
[
  {
    "event": {
      "Actor": {
        "username": "admin",
        "name": "admin",
        "email": "",
        "admin": true
      },
      "ActorType": "user",
      "Time": "2014-05-04T21:56:09.770856195-07:00",
      "Action": "create",
      "ObjectType": "*node.Node",
      "ObjectName": "mook",
      "ExtendedInfo": "{\"name\":\"mook\",\"chef_environment\":\"_default\",\"run_list\":[],\"json_class\":\"Chef::Node\",\"chef_type\":\"node\",\"automatic\":{},\"normal\":{},\"default\":{},\"override\":{}}\n",
      "Id": 1
    },
    "url": "http://terqa.local:4545/events/1"
  }
]
```

### Tested Platforms

Goiardi has been built and run with the native 6g compiler on Mac OS X (10.7,
10.8, and 10.9), Debian squeeze and wheezy, a fairly recent Arch Linux, and 
FreeBSD 9.2.

Goiardi has also been built and run with gccgo (using the `-compiler gccgo`
option with the `go` command) on Arch Linux. Building it with gccgo without 
the go command probably works, but it hasn't happened yet. This is a priority,
though, so goiardi can be built on platforms the native compiler doesn't support
yet.

### Note regarding goiardi persistence and freezing data

As mentioned above, goiardi can now freeze its in-memory data store and index to
disk if specified. It will save before quitting if the program receives a 
SIGTERM or SIGINT signal, along with saving every "freeze-interval" seconds
automatically.

Saving automatically helps guard against the case where the server receives a 
signal that it can't handle and forces it to quit. In addition, goiardi will not
replace the old save files until the new one is all finished writing. However,
it's still not anywhere near a real database with transaction protection, etc.,
so while it should work fine in the general case, possibilities for data loss
and corruption do exist. The appropriate caution is warranted.

DOCUMENTATION
-------------
In addition to the aforementioned Chef documentation at http://docs.opscode.com,
more documentation specific to goiardi can be viewed with godoc. See 
http://godoc.org/code.google.com/p/go.tools/cmd/godoc for an explanation of how
godoc works. The goiardi godocs can also be viewed online at 
http://godoc.org/github.com/ctdk/goiardi.

TODO
----

See the TODO file for an up-to-date list of what needs to be done. There's a
lot.

BUGS
----

There's going to be a lot of these for a while, so we'll just keep those in a
BUGS file, won't we?

WHY?
----

This started as a project to learn Go, and because I thought that an in memory
chef server would be handy. Then I found out about chef-zero, but I still wanted
a project to learn Go, so I kept it up. Chef 11 Server also only runs under
Linux at this time, while Goiardi is developed under Mac OS X and ought to run
under any platform Go supports (only partially tested at this time though).

CONTRIBUTING
------------

If you feel like contributing, great! Just fork the repo, make your
improvements, and submit a pull request. Tests would, of course, be appreciated.
Adding tests where there are no tests currently would be even more appreciated.
At least, though, try and not break anything worse than it is. Test coverage has
improved, but is still an ongoing concern.
AUTHOR
------

Jeremy Bingham (<jbingham@gmail.com>)

COPYRIGHT
---------

Copyright 2013-2014, Jeremy Bingham

LICENSE
-------

Like many Chef ecosystem programs, goairdi is licensed under the Apache 2.0 
License. See the LICENSE file for details.

Chef is copyright (c) 2008-2013 Opscode, Inc. and its various contributors.

Thanks go out to the fine folks of Opscode and the Chef community for all their
hard work.

Also, if you were wondering, Ettore Boiardi was the man behind Chef Boyardee. 
Wakka wakka.
