/*
 * Copyright (c) 2013, Jeremy Bingham (<jbingham@gmail.com>)
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

/* 
Goiardi is a simple implementation of the Chef server (http://www.opscode.com) 
written in Go. It currently runs entirely in memory, and draws heavy inspiration
from chef-zero.

It is very much a work in progress. At the moment basic functionality as tested
with knife works, and chef-client runs complete successfully. It is far enough 
along to run chef-pendant tests, even though it fails most of them.

Goiardi currently only has three dependencies: go-flags, go-cache, and go-trie. 
To install them, run:

   go get github.com/jessevdk/go-flags
   go get github.com/pmylund/go-cache
   go get github.com/ctdk/go-trie/gtrie

from your $GOROOT.

If you would like to modify the search grammar, you'll need the 'peg' package.
To install that, run

   go get github.com/pointlander/peg

In the 'search/' directory, run 'peg -switch -inline search-parse.peg' to
generate the new grammar. If you don't plan on editing the search grammar,
though, you won't need that.

To install:

1. Install go. (http://golang.org/doc/install.html)

2. Make sure your $GOROOT and PATH are set up correctly per the Go installation
   instructions.

3. Download goairdi

   go get github.com/ctdk/goiardi

4. Run tests, as soon as there are tests to run.

5. Install the goiardi binaries.

   go install github.com/ctdk/goiardi

6. Run goiardi.

   goiardi <options>

   You can get a list of current options with the '-h' flag. Some of them may
   even work. As of this writing you can specify the hostname, IP address, and
   port goiardi uses and have it actually do something useful.

   Goiardi can also take a config file, run like goiardi -c /path/to/conf-file.    See etc/goiardi.conf-sample for an example configuration file. Currently
   "hostname", "ipaddress", and "port" can be configured in the conf file (one
   per line). Options specified on the command line override options in the 
   config file.

For more documentation on Chef, see http://docs.opscode.com.

Goiardi does not actually care about .pem files at all at the moment, but you
still need to have one to keep knife and chef-client happy. It's like chef-zero
in that regard.

Tested Platforms:

Goiardi has been built and run with the native 6g compiler on Mac OS X (10.7 and
10.8), Debian wheezy, and a fairly recent Arch Linux.

Goiardi has also been built and run with gccgo (using the "-compiler gccgo"
option with the "go" command) on Arch Linux. Building it with gccgo without 
the go command probably works, but it hasn't happened yet. This is a priority,
though, so goiardi can be built on platforms the native compiler doesn't support
yet.

To do:

See the TODO file for an up-to-date list of what needs to be done. There's a
lot.

Bugs:

There's going to be a lot of these for a while, so we'll just keep those in a
BUGS file, won't we?

Why?:

This started as a project to learn Go, and because I thought that an in memory
chef server would be handy. Then I found out about chef-zero, but I still wanted
a project to learn Go, so I kept it up. Chef 11 Server also only runs under
Linux at this time, while Goiardi is developed under Mac OS X and ought to run
under any platform Go supports (this is completely untested at this time
though).

If you feel like contributing, great! Just fork the repo, make your
improvements, and submit a pull request. Tests would, of course, be appreciated.
Adding tests where there are no tests currently would be even more appreciated.
At least, though, try and not break anything worse than it is. Test coverage is
an ongoing concern.

Goiardi is authored and copyright (c) Jeremy Bingham, 2013.  Like many Chef 
ecosystem programs, goairdi is licensed under the Apache 2.0 License. See the 
LICENSE file for details.

Chef is copyright (c) 2008-2013 Opscode, Inc. and its various contributors.

Thanks go out to the fine folks of Opscode and the Chef community for all their
hard work.

Also, if you were wondering, Ettore Boiardi was the man behind Chef Boyardee. Wakka wakka.

*/
package main
