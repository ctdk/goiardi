Goiardi
=======

Goiardi is an implementation of the Chef server (http://www.opscode.com) written
in Go. It currently runs entirely in memory with the option to save and load the
in-memory data to and from disk, and draws heavy inspiration from chef-zero.

It is very much a work in progress. At the moment basic functionality as tested
with knife works, and chef-client runs complete successfully. It is far enough 
along to run chef-pendant tests. The authentication and permissions tests from
chef-pedant all fail at this time, but the other relevant tests pass except for a few areas with disagreements about formatting.

Adding go tests is on the TODO list.

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

   You can get a list of current options with the '-h' flag. Some of them may
   even work. As of this writing you can specify the hostname, IP address, port,
   log, data, and index files, and the interval to save data to disk goiardi 
   uses.

   Goiardi can also take a config file, run like `goiardi -c 
   /path/to/conf-file`. See `etc/goiardi.conf-sample` for an example documented
   configuration file. Currently `hostname`, `ipaddress`, `port`, `log-file`,
   `data-file`, `index-file`, and `freeze-interval` can be configured in the 
   conf file (one per line). Options specified on the command line override 
   options in the config file.

For more documentation on Chef, see (http://docs.opscode.com).

Goiardi does not actually care about .pem files at all at the moment, but you
still need to have one to keep knife and chef-client happy. It's like chef-zero
in that regard.

### Tested Platforms

Goiardi has been built and run with the native 6g compiler on Mac OS X (10.7 and
10.8), Debian wheezy, a fairly recent Arch Linux, and FreeBSD 9.2.

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
