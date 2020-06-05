.. _installation:

Installation
============

To install goiardi from source:

1. Install go. (http://golang.org/doc/install.html) Goiardi now requires go 1.7+ (because of the use of contexts). Goiardi should generally be able to be built with the latest version of Go, and this is generally recommended. Usually it will also build with the previous minor release, and may build with older versions as well, but this shouldn't be relied on. Immediately after a minor release, of course, caution may be warranted.

2. Make sure your ``$GOROOT`` and ``$PATH`` are set up correctly per the Go installation instructions.

3. Download goairdi and its dependencies

    go get -t -u github.com/ctdk/goiardi

4. Run tests, if desired. Several goiardi subdirectories have go tests, and chef-pedant can and should be used for testing goiardi as well.

5. Install the goiardi binaries.

    go install github.com/ctdk/goiardi

6. Run goiardi.

    goiardi <options>

   Or, you can look at the goiardi releases page on github at https://github.com/ctdk/goiardi/releases and see if there are precompiled binaries available for your platform, or check out the packages at https://packagecloud.io/ct/goiardi and see if there's one for your platform there.

Another option is running goiardi in Docker. There's a Dockerfile in the root of the goiardi git repository that's suitable for running the local version of goiardi, but a goiardi repository on Docker Hub at https://hub.docker.com/r/ctdk/goiardi/ is also under development (the source repository for those docker images is at https://github.com/ctdk/goiardi-docker). Running goiardi under docker has always worked fine, but now that configuration options can be set with environment variables it's certainly easier to do so than before.

Configuration
=============

You can get a list of command-line options with the ``-h`` flag.

Additionally, many of goiardi's options that can be set with flags can also be set with environment variables. Where this is the case, the option's description will be followed by an environment variable name (like ``$GOIARDI_HANDY_OPTION``).

Goiardi can also take a config file, run like ``goiardi -c /path/to/conf-file``. See ``etc/goiardi.conf-sample`` for an example documented configuration file. Options in the configuration file share the same name as the long command line arguments (so, for example, ``--ipaddress=127.0.0.1`` on the command line would be ``ipaddress = "127.0.0.1"`` in the config file.

Currently available command line and config file options::

    -v, --version               Print version info.
    -V, --verbose               Show verbose debug information. Repeat for more
                                verbosity.
    -c, --config=               Specify a config file to use. [$GOIARDI_CONFIG]
    -I, --ipaddress=            Listen on a specific IP address.
                                [$GOIARDI_IPADDRESS]
    -H, --hostname=             Hostname to use for this server. Defaults to
                                hostname reported by the kernel.
                                [$GOIARDI_HOSTNAME]
    -P, --port=                 Port to listen on. If port is set to 443, SSL
                                will be activated. (default: 4545) [$GOIARDI_PORT]
    -Z, --proxy-hostname=       Hostname to report to clients if this goiardi
                                server is behind a proxy using a different
                                hostname. See also --proxy-port. Can be used with
                                --proxy-port or alone, or not at all.
                                [$GOIARDI_PROXY_HOSTNAME]
    -W, --proxy-port=           Port to report to clients if this goiardi server
                                is behind a proxy using a different port than the
                                port goiardi is listening on. Can be used with
                                --proxy-hostname or alone, or not at all.
                                [$GOIARDI_PROXY_PORT]
    -i, --index-file=           File to save search index data to.
                                [$GOIARDI_INDEX_FILE]
    -D, --data-file=            File to save data store data to.
                                [$GOIARDI_DATA_FILE]
    -F, --freeze-interval=      Interval in seconds to freeze in-memory data
                                structures to disk if there have been any changes
                                (requires -i/--index-file and -D/--data-file
                                options to be set). (Default 10 seconds.)
                                [$GOIARDI_FREEZE_INTERVAL]
    -L, --log-file=             Log to file X [$GOIARDI_LOG_FILE]
    -s, --syslog                Log to syslog rather than a log file.
                                Incompatible with -L/--log-file. [$GOIARDI_SYSLOG]
    -g, --log-level=            Specify logging verbosity. Performs the same
                                function as -V, but works like the 'log-level'
                                option in the configuration file. Acceptable
                                values are 'debug', 'info', 'warning', 'error',
                                'critical', and 'fatal'. [$GOIARDI_LOG_LEVEL]
        --time-slew=            Time difference allowed between the server's
                                clock and the time in the X-OPS-TIMESTAMP header.
                                Formatted like 5m, 150s, etc. Defaults to 15m.
                                [$GOIARDI_TIME_SLEW]
        --conf-root=            Root directory for configs and certificates.
                                Default: the directory the config file is in, or
                                the current directory if no config file is set.
                                [$GOIARDI_CONF_ROOT]
    -A, --use-auth              Use authentication. Default: false. (NB: At a
                                future time, the default behavior will change to
                                authentication being enabled.) [$GOIARDI_USE_AUTH]
        --use-ssl               Use SSL for connections. If --port is set to 433,
                                this will automatically be turned on. If it is
                                set to 80, it will automatically be turned off.
                                Default: off. Requires --ssl-cert and --ssl-key.
                                [$GOIARDI_USE_SSL]
        --ssl-cert=             SSL certificate file. If a relative path, will be
                                set relative to --conf-root. [$GOIARDI_SSL_CERT]
        --ssl-key=              SSL key file. If a relative path, will be set
                                relative to --conf-root. [$GOIARDI_SSL_KEY]
        --https-urls            Use 'https://' in URLs to server resources if
                                goiardi is not using SSL for its connections.
                                Useful when goiardi is sitting behind a reverse
                                proxy that uses SSL, but is communicating with
                                the proxy over HTTP. [$GOIARDI_HTTPS_URLS]
        --disable-webui         If enabled, disables connections and logins to
                                goiardi over the webui interface.
                                [$GOIARDI_DISABLE_WEBUI]
        --use-mysql             Use a MySQL database for data storage. Configure
                                database options in the config file.
                                [$GOIARDI_USE_MYSQL]
        --use-postgresql        Use a PostgreSQL database for data storage.
                                Configure database options in the config file.
                                [$GOIARDI_USE_POSTGRESQL]
        --local-filestore-dir=  Directory to save uploaded files in. Optional
                                when running in in-memory mode, *mandatory*
                                (unless using S3 uploads) for SQL mode.
                                [$GOIARDI_LOCAL_FILESTORE_DIR]
        --log-events            Log changes to chef objects. [$GOIARDI_LOG_EVENTS]
    -K, --log-event-keep=       Number of events to keep in the event log. If
                                set, the event log will be checked periodically
                                and pruned to this number of entries.
                                [$GOIARDI_LOG_EVENT_KEEP]
        --skip-log-extended     If set, do not save a JSON encoded blob of the
                                object being logged when logging an event.
                                [$GOIARDI_SKIP_LOG_EXTENDED]
    -x, --export=               Export all server data to the given file, exiting
                                afterwards. Should be used with caution. Cannot
                                be used at the same time as -m/--import.
    -m, --import=               Import data from the given file, exiting
                                afterwards. Cannot be used at the same time as
                                -x/--export.
        --bootstrap             Initialize server clients and admin user.
                                Exits with status code 0 if everything went ok.
    -Q, --obj-max-size=         Maximum object size in bytes for the file store.
                                Default 10485760 bytes (10MB).
                                [$GOIARDI_OBJ_MAX_SIZE]
    -j, --json-req-max-size=    Maximum size for a JSON request from the client.
                                Per chef-pedant, default is 1000000.
                                [$GOIARDI_JSON_REQ_MAX_SIZE]
        --use-unsafe-mem-store  Use the faster, but less safe, old method of
                                storing data in the in-memory data store with
                                pointers, rather than encoding the data with gob
                                and giving a new copy of the object to each
                                requestor. If this is enabled goiardi will run
                                faster in in-memory mode, but one goroutine could
                                change an object while it's being used by
                                another. Has no effect when using an SQL backend.
                                (DEPRECATED - will be removed in a future
                                release.)
        --db-pool-size=         Number of idle db connections to maintain. Only
                                useful when using one of the SQL backends.
                                Default is 0 - no idle connections retained
                                [$GOIARDI_DB_POOL_SIZE]
        --max-connections=      Maximum number of connections allowed for the
                                database. Only useful when using one of the SQL
                                backends. Default is 0 - unlimited.
                                [$GOIARDI_MAX_CONN]
        --use-serf              If set, have goidari use serf to send and receive
                                events and queries from a serf cluster. Required
                                for shovey. [$GOIARDI_USE_SERF]
        --serf-event-announce   Announce log events and joining the serf cluster
                                over serf, as serf events. Requires --use-serf.
                                [$GOIARDI_SERF_EVENT_ANNOUNCE]
        --serf-addr=            IP address and port to use for RPC communication
                                with a serf agent. Defaults to 127.0.0.1:7373.
                                [$GOIARDI_SERF_ADDR]
        --use-shovey            Enable using shovey for sending jobs to nodes.
                                Requires --use-serf. [$GOIARDI_USE_SHOVEY]
        --sign-priv-key=        Path to RSA private key used to sign shovey
                                requests. [$GOIARDI_SIGN_PRIV_KEY]
        --dot-search            If set, searches will use . to separate elements
                                instead of _. [$GOIARDI_DOT_SEARCH]
        --convert-search        If set, convert _ syntax searches to . syntax.
                                Only useful if --dot-search is set.
                                [$GOIARDI_CONVERT_SEARCH]
        --pg-search             Use the new Postgres based search engine instead
                                of the default ersatz Solr. Requires
                                --use-postgresql, automatically turns on
                                --dot-search. --convert-search is recommended,
                                but not required. [$GOIARDI_PG_SEARCH]
        --use-statsd            Whether or not to collect statistics about
                                goiardi and send them to statsd.
                                [$GOIARDI_USE_STATSD]
        --statsd-addr=          IP address and port of statsd instance to connect
                                to. (default 'localhost:8125')
                                [$GOIARDI_STATSD_ADDR]
        --statsd-type=          statsd format, can be either 'standard' or
                                'datadog' (default 'standard')
                                [$GOIARDI_STATSD_TYPE]
        --statsd-instance=      Statsd instance name to use for this server.
                                Defaults to the server's hostname, with '.'
                                replaced by '_'. [$GOIARDI_STATSD_INSTANCE]
        --use-s3-upload         Store cookbook files in S3 rather than locally in
                                memory or on disk. This or --local-filestore-dir
                                must be set in SQL mode. Cannot be used with
                                in-memory mode. [$GOIARDI_USE_S3_UPLOAD]
        --aws-region=           AWS region to use S3 uploads.
                                [$GOIARDI_AWS_REGION]
        --s3-bucket=            The name of the S3 bucket storing the files.
                                [$GOIARDI_S3_BUCKET]
        --aws-disable-ssl       Set to disable SSL for the endpoint. Mostly
                                useful just for testing.
                                [$GOIARDI_AWS_DISABLE_SSL]
        --s3-endpoint=          Set a different endpoint than the default
                                s3.amazonaws.com. Mostly useful for testing with
                                a fake S3 service, or if using an S3-compatible
                                service. [$GOIARDI_S3_ENDPOINT]
        --s3-file-period=       Length of time, in minutes, to allow files to be
                                saved to or retrieved from S3 by the client.
                                Defaults to 15 minutes. [$GOIARDI_S3_FILE_PERIOD]
        --use-external-secrets  Use an external service to store secrets
                                (currently user/client public keys). Currently
                                only vault is supported.
                                [$GOIARDI_USE_EXTERNAL_SECRETS]
        --vault-addr=           Specify address of vault server (i.e.
                                https://127.0.0.1:8200). Defaults to the value of
                                VAULT_ADDR.
        --vault-shovey-key=     Specify a path in vault holding shovey's private
                                key. The key must be put in vault as
                                'privateKey=<contents>'.
                                [$GOIARDI_VAULT_SHOVEY_KEY]
    -T, --index-val-trim=       Trim values indexed for chef search to this many
                                characters (keys are untouched). If not set or
                                set <= 0, trimming is disabled. This behavior
                                will change with the next major release.
                                [$GOIARDI_INDEX_VAL_TRIM]
    -y, --pprof-whitelist=      Address to allow to access /debug/pprof (in
                                addition to localhost). Specify multiple times to
                                allow more addresses. [$GOIARDI_PPROF_WHITELIST]
        --purge-reports-after=  Time to purge old reports after, given in golang
                                duration format (e.g. "720h"). Default is not to
                                purge them at all. [$GOIARDI_PURGE_REPORTS_AFTER]
        --purge-status-after=   Time to purge old node statuses after, given in
                                golang duration format (e.g. "720h"). Default is
                                not to purge them at all.
                                [$GOIARDI_PURGE_STATUS_AFTER]
        --purge-sandboxes-after= Time to purge old reports after, given in golang
                                duration format (e.g. "720h"). Default is to
                                purge them after one week. Set this to '0s' to
                                disable sandbox purging.
                                [$GOIARDI_PURGE_SANDBOXES_AFTER]

  MySQL connection options (requires --use-mysql):
        --mysql-username=       MySQL username [$GOIARDI_MYSQL_USERNAME]
        --mysql-password=       MySQL password [$GOIARDI_MYSQL_PASSWORD]
        --mysql-protocol=       MySQL protocol (tcp or unix)
                                [$GOIARDI_MYSQL_PROTOCOL]
        --mysql-address=        MySQL IP address, hostname, or path to a socket
                                [$GOIARDI_MYSQL_ADDRESS]
        --mysql-port=           MySQL TCP port [$GOIARDI_MYSQL_PORT]
        --mysql-dbname=         MySQL database name [$GOIARDI_MYSQL_DBNAME]
        --mysql-extra-params=   Extra configuration parameters for MySQL. Specify
                                them like '--mysql-extra-params=foo:bar'.
                                Multiple extra parameters can be specified by
                                supplying the --mysql-extra-params flag multiple
                                times. If using an environment variable, split up
                                multiple parameters with #, like so:
                                GOIARDI_MYSQL_EXTRA_PARAMS='foo:bar#baz:bug'.
                                [$GOIARDI_MYSQL_EXTRA_PARAMS]

  PostgreSQL connection options (requires --use-postgresql):
        --postgresql-username=  PostgreSQL user name
                                [$GOIARDI_POSTGRESQL_USERNAME]
        --postgresql-password=  PostgreSQL password [$GOIARDI_POSTGRESQL_PASSWORD]
        --postgresql-host=      PostgreSQL IP host, hostname, or path to a socket
                                [$GOIARDI_POSTGRESQL_HOST]
        --postgresql-port=      PostgreSQL TCP port [$GOIARDI_POSTGRESQL_PORT]
        --postgresql-dbname=    PostgreSQL database name
                                [$GOIARDI_POSTGRESQL_DBNAME]
        --postgresql-ssl-mode=  PostgreSQL SSL mode ('enable' or 'disable')
                                [$GOIARDI_POSTGRESQL_SSL_MODE]

**NB:** If goiardi has been compiled with the ``novault`` build tag, the help output will be missing ``--use-external-secrets``, ``--vault-addr``, and ``--vault-shovey-key``.

Options specified on the command line override options in the config file. Options specified via the command line override options in the config file, but are themselves overridden by command line flags.

For more documentation on Chef, see http://docs.chef.io.

Binaries and Packages
=====================

There are other options for installing goiardi, in case you don't want to build it from scratch. Binaries for several platforms are provided with each release, and there are .debs available as well at https://packagecloud.io/ct/goiardi. At the moment packages are being built for Debian wheezy and later, Ubuntu 14.04 and later current and upcoming releases, raspbian (which is under the Debian versions) for various Raspberry Pi computers, and CentOS 6 and 7. Packages for other platforms may happen down the road. As of this writing, debs for goiardi 0.11.2 can be `found in Debian stretch (a.k.a stable) <https://packages.qa.debian.org/g/goiardi.html>`_. More current versions of goiardi can be found in Debian's ``testing`` and ``unstable`` branches as well as in Ubuntu's ``universe`` repository since "Zesty Zapus".

**NB:** `wheezy` is currently (as of this writing) supported by the `Debian LTS <https://wiki.debian.org/LTS>`_ project. Sometime after that ends, which is scheduled for May 31st, 2018, it'll be dropped from the packagecloud.io builds and the supporting files removed from the repository.

There is also a `homebrew tap <https://github.com/ctdk/homebrew-ctdk>`_ that includes goiardi now, for folks running Mac OS X and using homebrew.
