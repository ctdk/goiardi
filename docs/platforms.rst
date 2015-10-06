.. _platforms:

Supported Platforms
===================

Goiardi has been built and run with the native 6g compiler on Mac OS X (10.7, 10.8, and 10.9), Debian squeeze and wheezy, a fairly recent Arch Linux, FreeBSD 9.2, Ubuntu 14.04, and Solaris. Using Go's cross compiling capabilities, goiardi builds for all of Go's supported platforms except plan9 (because of issues with the postgres client library). Windows support has not been tested extensively, but a cross compiled binary has been tested successfully on Windows.

Goiardi has also been built and run with gccgo (using the ``-compiler gccgo`` option with the ``go`` command) on Arch Linux. Building it with gccgo without the go command probably works, but it hasn't happened yet. This is a priority, though, so goiardi can be built on platforms the native compiler doesn't support yet.
