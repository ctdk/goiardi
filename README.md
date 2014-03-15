Goiardi
=======

Goiardi is an implementation of the Chef server (http://www.opscode.com) written
in Go. It currently runs entirely in memory with the option to save and load the
in-memory data to and from disk, and draws heavy inspiration from chef-zero.

It is a work in progress. At the moment normal functionality as tested with 
knife works, and chef-client runs complete successfully. It is far enough along 
to run most chef-pendant tests successfully. It does pretty well against the
official chef-pedant, but because goiardi handles some authentication matters a
little differently than the official chef-server, there is also a fork of
chef-pedant located at https://github.com/ctdk/chef-pedant that's more custom
tailored to goiardi.

Many go tests are present as well in different goiardi subdirectories.

DEPENDENCIES
------------

Goiardi currently has four dependencies: go-flags, go-cache, go-trie, and toml.
To install them, run:

```
   go get github.com/jessevdk/go-flags
   go get github.com/pmylund/go-cache
   go get github.com/ctdk/go-trie/gtrie
   go get github.com/BurntSushi/toml
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

4. Run tests, as soon as there are tests to run.

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
   -V, --verbose          Show verbose debug information. (not implemented)
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

### Tested Platforms

Goiardi has been built and run with the native 6g compiler on Mac OS X (10.7 and
10.8), Debian squeeze and wheezy, a fairly recent Arch Linux, and FreeBSD 9.2.

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
godoc works.

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
At least, though, try and not break anything worse than it is. Test coverage is
an ongoing concern.

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
