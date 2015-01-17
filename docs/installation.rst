.. _installation:

Installation
============


1. Install go. (http://golang.org/doc/install.html) You will need to use at least go 1.3 to compile goiardi, but go 1.4 works as well and is recommended.

2. Make sure your ``$GOROOT`` and ``$PATH`` are set up correctly per the Go installation instructions.

3. Download goairdi and its dependencies

    go get -t -u github.com/ctdk/goiardi

4. Run tests, if desired. Several goiardi subdirectories have go tests, and chef-pedant can and should be used for testing goiardi as well.

5. Install the goiardi binaries.

    go install github.com/ctdk/goiardi

6. Run goiardi.

    goiardi <options>

   Or, you can look at the goiardi releases page on github at https://github.com/ctdk/goiardi/releases and see if there are precompiled binaries available for your platform.

You can get a list of command-line options with the ``-h`` flag.

Goiardi can also take a config file, run like ``goiardi -c /path/to/conf-file``. See ``etc/goiardi.conf-sample`` for an example documented configuration file. Options in the configuration file share the same name as the long command line arguments (so, for example, ``--ipaddress=127.0.0.1`` on the command line would be ``ipaddress = "127.0.0.1"`` in the config file.

Currently available command line and config file options::

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
                           -D/--data-file options to be set). (Default 10
                           seconds.)
    -L, --log-file=        Log to file X
    -s, --syslog           Log to syslog rather than a log file. Incompatible
                           with -L/--log-file.
        --time-slew=       Time difference allowed between the server's clock and
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
        --use-postgresql   Use a PostgreSQL database for data storage.
                           Configure database options in the config file.
        --local-filestore-dir= Directory to save uploaded files in. Optional when
                           running in in-memory mode, *mandatory* for SQL
                           mode.
        --log-events       Log changes to chef objects.
    -K, --log-event-keep=  Number of events to keep in the event log. If set,
                           the event log will be checked periodically and
                           pruned to this number of entries.
    -x, --export=          Export all server data to the given file, exiting
                           afterwards. Should be used with caution. Cannot be
                           used at the same time as -m/--import.
    -m, --import=          Import data from the given file, exiting
                           afterwards. Cannot be used at the same time as
                           -x/--export.
    -Q, --obj-max-size=    Maximum object size in bytes for the file store.
                           Default 10485760 bytes (10MB).
    -j, --json-req-max-size= Maximum size for a JSON request from the client.
                           Per chef-pedant, default is 1000000.
        --use-unsafe-mem-store Use the faster, but less safe, old method of
                           storing data in the in-memory data store with
                           pointers, rather than encoding the data with gob
                           and giving a new copy of the object to each
                           requestor. If this is enabled goiardi will run
                           faster in in-memory mode, but one goroutine could
                           change an object while it's being used by
                           another. Has no effect when using an SQL backend.
        --db-pool-size=    Number of idle db connections to maintain. Only
                           useful when using one of the SQL backends.
                           Default is 0 - no idle connections retained
        --max-connections= Maximum number of connections allowed for the
                           database. Only useful when using one of the SQL
                           backends. Default is 0 - unlimited.
        --use-serf         If set, have goidari use serf to send and receive
                           events and queries from a serf cluster. Required
                           for shovey.
        --serf-event-announce Announce log events over serf and joining the serf
                           cluster, as serf events. Requires --use-serf.
        --serf-addr=       IP address and port to use for RPC communication
                           with a serf agent. Defaults to 127.0.0.1:7373.
        --use-shovey       Enable using shovey for sending jobs to nodes.
               Requires --use-serf.
        --sign-priv-key=   Path to RSA private key used to sign shovey
                           requests.

Options specified on the command line override options in the config file.

For more documentation on Chef, see (http://docs.chef.io).
