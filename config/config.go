/* Goiardi configuration. */

/*
 * Copyright (c) 2013-2016, Jeremy Bingham (<jeremy@goiardi.gl>)
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

// Package config parses command line flags and config files, and defines
// options used elsewhere in goiardi.
package config

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/jessevdk/go-flags"
	"github.com/tideland/golib/logger"
)

// Conf is the master struct for holding configuration options.
type Conf struct {
	Ipaddress         string
	Port              int
	Hostname          string
	ProxyHostname     string `toml:"proxy-hostname"`
	ProxyPort         int    `toml:"proxy-port"`
	ConfFile          string `toml:"conf-file"`
	IndexFile         string `toml:"index-file"`
	DataStoreFile     string `toml:"data-file"`
	DebugLevel        int    `toml:"debug-level"`
	LogLevel          string `toml:"log-level"`
	FreezeInterval    int    `toml:"freeze-interval"`
	FreezeData        bool   `toml:"freeze-data"`
	LogFile           string `toml:"log-file"`
	SysLog            bool   `toml:"syslog"`
	UseAuth           bool   `toml:"use-auth"`
	TimeSlew          string `toml:"time-slew"`
	TimeSlewDur       time.Duration
	ConfRoot          string       `toml:"conf-root"`
	UseSSL            bool         `toml:"use-ssl"`
	SSLCert           string       `toml:"ssl-cert"`
	SSLKey            string       `toml:"ssl-key"`
	HTTPSUrls         bool         `toml:"https-urls"`
	DisableWebUI      bool         `toml:"disable-webui"`
	UseMySQL          bool         `toml:"use-mysql"`
	MySQL             MySQLdb      `toml:"mysql"`
	UsePostgreSQL     bool         `toml:"use-postgresql"`
	PostgreSQL        PostgreSQLdb `toml:"postgresql"`
	LocalFstoreDir    string       `toml:"local-filestore-dir"`
	LogEvents         bool         `toml:"log-events"`
	LogEventKeep      int          `toml:"log-event-keep"`
	DoExport          bool
	DoImport          bool
	ImpExFile         string
	ObjMaxSize        int64    `toml:"obj-max-size"`
	JSONReqMaxSize    int64    `toml:"json-req-max-size"`
	UseUnsafeMemStore bool     `toml:"use-unsafe-mem-store"`
	DbPoolSize        int      `toml:"db-pool-size"`
	MaxConn           int      `toml:"max-connections"`
	UseSerf           bool     `toml:"use-serf"`
	SerfEventAnnounce bool     `toml:"serf-event-announce"`
	SerfAddr          string   `toml:"serf-addr"`
	UseShovey         bool     `toml:"use-shovey"`
	SignPrivKey       string   `toml:"sign-priv-key"`
	DotSearch         bool     `toml:"dot-search"`
	ConvertSearch     bool     `toml:"convert-search"`
	PgSearch          bool     `toml:"pg-search"`
	UseStatsd         bool     `toml:"use-statsd"`
	StatsdAddr        string   `toml:"statsd-addr"`
	StatsdType        string   `toml:"statsd-type"`
	StatsdInstance    string   `toml:"statsd-instance"`
	UseS3Upload       bool     `toml:"use-s3-upload"`
	AWSRegion         string   `toml:"aws-region"`
	S3Bucket          string   `toml:"s3-bucket"`
	AWSDisableSSL     bool     `toml:"aws-disable-ssl"`
	S3Endpoint        string   `toml:"s3-endpoint"`
	S3FilePeriod      int      `toml:"s3-file-period"`
	UseExtSecrets     bool     `toml:"use-external-secrets"`
	VaultAddr         string   `toml:"vault-addr"`
	VaultShoveyKey    string   `toml:"vault-shovey-key"`
	EnvVars           []string `toml:"env-vars"`
	IndexValTrim      int      `toml:"index-val-trim"`
}

// SigningKeys are the public and private keys for signing shovey requests.
type SigningKeys struct {
	sync.RWMutex
	PrivKey *rsa.PrivateKey
}

// Key is the initialized shovey public and private keys.
var Key = &SigningKeys{}

// GitHash is the git hash (supplied with '-ldflags "-X config.GitHash=<hash>"')
// of goiardi when it was compiled.
var GitHash = "unknown"

// LogLevelNames give convenient, easier to remember than number name for the
// different levels of logging.
var LogLevelNames = map[string]int{"debug": 5, "info": 4, "warning": 3, "error": 2, "critical": 1, "fatal": 0}

// MySQLdb holds MySQL connection options.
type MySQLdb struct {
	Username    string            `long:"username" description:"MySQL username" env:"GOIARDI_MYSQL_USERNAME"`
	Password    string            `long:"password" description:"MySQL password" env:"GOIARDI_MYSQL_PASSWORD"`
	Protocol    string            `long:"protocol" description:"MySQL protocol (tcp or unix)" env:"GOIARDI_MYSQL_PROTOCOL"`
	Address     string            `long:"address" description:"MySQL IP address, hostname, or path to a socket" env:"GOIARDI_MYSQL_ADDRESS"`
	Port        string            `long:"port" description:"MySQL TCP port" env:"GOIARDI_MYSQL_PORT"`
	Dbname      string            `long:"dbname" description:"MySQL database name" env:"GOIARDI_MYSQL_DBNAME"`
	ExtraParams map[string]string `toml:"extra_params" long:"extra-params" description:"Extra configuration parameters for MySQL. Specify them like '--mysql-extra-params=foo:bar'. Multiple extra parameters can be specified by supplying the --mysql-extra-params flag multiple times. If using an environment variable, split up multiple parameters with #, like so: GOIARDI_MYSQL_EXTRA_PARAMS='foo:bar#baz:bug'." env:"GOIARDI_MYSQL_EXTRA_PARAMS" env-delim:"#"`
}

// PostgreSQLdb holds Postgres connection options.
type PostgreSQLdb struct {
	Username string `long:"username" description:"PostgreSQL user name" env:"GOIARDI_POSTGRESQL_USERNAME"`
	Password string `long:"password" description:"PostgreSQL password" env:"GOIARDI_POSTGRESQL_PASSWORD"`
	Host     string `long:"host" description:"PostgreSQL IP host, hostname, or path to a socket" env:"GOIARDI_POSTGRESQL_HOST"`
	Port     string `long:"port" description:"PostgreSQL TCP port" env:"GOIARDI_POSTGRESQL_PORT"`
	Dbname   string `long:"dbname" description:"PostgreSQL database name" env:"GOIARDI_POSTGRESQL_DBNAME"`
	SSLMode  string `long:"ssl-mode" description:"PostgreSQL SSL mode ('enable' or 'disable')" env:"GOIARDI_POSTGRESQL_SSL_MODE"`
}

// Options holds options set from the command line or (in most cases)
// environment variables, which are then merged with the options in Conf.
// Configurations from the command line/env vars are preferred to those set in
// the config file.
type Options struct {
	Version           bool         `short:"v" long:"version" description:"Print version info."`
	Verbose           []bool       `short:"V" long:"verbose" description:"Show verbose debug information. Repeat for more verbosity."`
	ConfFile          string       `short:"c" long:"config" description:"Specify a config file to use." env:"GOIARDI_CONFIG"`
	Ipaddress         string       `short:"I" long:"ipaddress" description:"Listen on a specific IP address." env:"GOIARDI_IPADDRESS"`
	Hostname          string       `short:"H" long:"hostname" description:"Hostname to use for this server. Defaults to hostname reported by the kernel." env:"GOIARDI_HOSTNAME"`
	Port              int          `short:"P" long:"port" description:"Port to listen on. If port is set to 443, SSL will be activated. (default: 4545)" env:"GOIARDI_PORT"`
	ProxyHostname     string       `short:"Z" long:"proxy-hostname" description:"Hostname to report to clients if this goiardi server is behind a proxy using a different hostname. See also --proxy-port. Can be used with --proxy-port or alone, or not at all." env:"GOIARDI_PROXY_HOSTNAME"`
	ProxyPort         int          `short:"W" long:"proxy-port" description:"Port to report to clients if this goiardi server is behind a proxy using a different port than the port goiardi is listening on. Can be used with --proxy-hostname or alone, or not at all." env:"GOIARDI_PROXY_PORT"`
	IndexFile         string       `short:"i" long:"index-file" description:"File to save search index data to." env:"GOIARDI_INDEX_FILE"`
	DataStoreFile     string       `short:"D" long:"data-file" description:"File to save data store data to." env:"GOIARDI_DATA_FILE"`
	FreezeInterval    int          `short:"F" long:"freeze-interval" description:"Interval in seconds to freeze in-memory data structures to disk if there have been any changes (requires -i/--index-file and -D/--data-file options to be set). (Default 10 seconds.)" env:"GOIARDI_FREEZE_INTERVAL"`
	LogFile           string       `short:"L" long:"log-file" description:"Log to file X" env:"GOIARDI_LOG_FILE"`
	SysLog            bool         `short:"s" long:"syslog" description:"Log to syslog rather than a log file. Incompatible with -L/--log-file." env:"GOIARDI_SYSLOG"`
	LogLevel          string       `short:"g" long:"log-level" description:"Specify logging verbosity. Performs the same function as -V, but works like the 'log-level' option in the configuration file. Acceptable values are 'debug', 'info', 'warning', 'error', 'critical', and 'fatal'." env:"GOIARDI_LOG_LEVEL"`
	TimeSlew          string       `long:"time-slew" description:"Time difference allowed between the server's clock and the time in the X-OPS-TIMESTAMP header. Formatted like 5m, 150s, etc. Defaults to 15m." env:"GOIARDI_TIME_SLEW"`
	ConfRoot          string       `long:"conf-root" description:"Root directory for configs and certificates. Default: the directory the config file is in, or the current directory if no config file is set." env:"GOIARDI_CONF_ROOT"`
	UseAuth           bool         `short:"A" long:"use-auth" description:"Use authentication. Default: false. (NB: At a future time, the default behavior will change to authentication being enabled.)" env:"GOIARDI_USE_AUTH"`
	UseSSL            bool         `long:"use-ssl" description:"Use SSL for connections. If --port is set to 433, this will automatically be turned on. If it is set to 80, it will automatically be turned off. Default: off. Requires --ssl-cert and --ssl-key." env:"GOIARDI_USE_SSL"`
	SSLCert           string       `long:"ssl-cert" description:"SSL certificate file. If a relative path, will be set relative to --conf-root." env:"GOIARDI_SSL_CERT"`
	SSLKey            string       `long:"ssl-key" description:"SSL key file. If a relative path, will be set relative to --conf-root." env:"GOIARDI_SSL_KEY"`
	HTTPSUrls         bool         `long:"https-urls" description:"Use 'https://' in URLs to server resources if goiardi is not using SSL for its connections. Useful when goiardi is sitting behind a reverse proxy that uses SSL, but is communicating with the proxy over HTTP." env:"GOIARDI_HTTPS_URLS"`
	DisableWebUI      bool         `long:"disable-webui" description:"If enabled, disables connections and logins to goiardi over the webui interface." env:"GOIARDI_DISABLE_WEBUI"`
	UseMySQL          bool         `long:"use-mysql" description:"Use a MySQL database for data storage. Configure database options in the config file." env:"GOIARDI_USE_MYSQL"`
	MySQL             MySQLdb      `group:"MySQL connection options (requires --use-mysql)" namespace:"mysql"`
	UsePostgreSQL     bool         `long:"use-postgresql" description:"Use a PostgreSQL database for data storage. Configure database options in the config file." env:"GOIARDI_USE_POSTGRESQL"`
	PostgreSQL        PostgreSQLdb `group:"PostgreSQL connection options (requires --use-postgresql)" namespace:"postgresql"`
	LocalFstoreDir    string       `long:"local-filestore-dir" description:"Directory to save uploaded files in. Optional when running in in-memory mode, *mandatory* (unless using S3 uploads) for SQL mode." env:"GOIARDI_LOCAL_FILESTORE_DIR"`
	LogEvents         bool         `long:"log-events" description:"Log changes to chef objects." env:"GOIARDI_LOG_EVENTS"`
	LogEventKeep      int          `short:"K" long:"log-event-keep" description:"Number of events to keep in the event log. If set, the event log will be checked periodically and pruned to this number of entries." env:"GOIARDI_LOG_EVENT_KEEP"`
	Export            string       `short:"x" long:"export" description:"Export all server data to the given file, exiting afterwards. Should be used with caution. Cannot be used at the same time as -m/--import."`
	Import            string       `short:"m" long:"import" description:"Import data from the given file, exiting afterwards. Cannot be used at the same time as -x/--export."`
	ObjMaxSize        int64        `short:"Q" long:"obj-max-size" description:"Maximum object size in bytes for the file store. Default 10485760 bytes (10MB)." env:"GOIARDI_OBJ_MAX_SIZE"`
	JSONReqMaxSize    int64        `short:"j" long:"json-req-max-size" description:"Maximum size for a JSON request from the client. Per chef-pedant, default is 1000000." env:"GOIARDI_JSON_REQ_MAX_SIZE"`
	UseUnsafeMemStore bool         `long:"use-unsafe-mem-store" description:"Use the faster, but less safe, old method of storing data in the in-memory data store with pointers, rather than encoding the data with gob and giving a new copy of the object to each requestor. If this is enabled goiardi will run faster in in-memory mode, but one goroutine could change an object while it's being used by another. Has no effect when using an SQL backend. (DEPRECATED - will be removed in a future release.)"`
	DbPoolSize        int          `long:"db-pool-size" description:"Number of idle db connections to maintain. Only useful when using one of the SQL backends. Default is 0 - no idle connections retained" env:"GOIARDI_DB_POOL_SIZE"`
	MaxConn           int          `long:"max-connections" description:"Maximum number of connections allowed for the database. Only useful when using one of the SQL backends. Default is 0 - unlimited." env:"GOIARDI_MAX_CONN"`
	UseSerf           bool         `long:"use-serf" description:"If set, have goidari use serf to send and receive events and queries from a serf cluster. Required for shovey." env:"GOIARDI_USE_SERF"`
	SerfEventAnnounce bool         `long:"serf-event-announce" description:"Announce log events and joining the serf cluster over serf, as serf events. Requires --use-serf." env:"GOIARDI_SERF_EVENT_ANNOUNCE"`
	SerfAddr          string       `long:"serf-addr" description:"IP address and port to use for RPC communication with a serf agent. Defaults to 127.0.0.1:7373." env:"GOIARDI_SERF_ADDR"`
	UseShovey         bool         `long:"use-shovey" description:"Enable using shovey for sending jobs to nodes. Requires --use-serf." env:"GOIARDI_USE_SHOVEY"`
	SignPrivKey       string       `long:"sign-priv-key" description:"Path to RSA private key used to sign shovey requests." env:"GOIARDI_SIGN_PRIV_KEY"`
	DotSearch         bool         `long:"dot-search" description:"If set, searches will use . to separate elements instead of _." env:"GOIARDI_DOT_SEARCH"`
	ConvertSearch     bool         `long:"convert-search" description:"If set, convert _ syntax searches to . syntax. Only useful if --dot-search is set." env:"GOIARDI_CONVERT_SEARCH"`
	PgSearch          bool         `long:"pg-search" description:"Use the new Postgres based search engine instead of the default ersatz Solr. Requires --use-postgresql, automatically turns on --dot-search. --convert-search is recommended, but not required." env:"GOIARDI_PG_SEARCH"`
	UseStatsd         bool         `long:"use-statsd" description:"Whether or not to collect statistics about goiardi and send them to statsd." env:"GOIARDI_USE_STATSD"`
	StatsdAddr        string       `long:"statsd-addr" description:"IP address and port of statsd instance to connect to. (default 'localhost:8125')" env:"GOIARDI_STATSD_ADDR"`
	StatsdType        string       `long:"statsd-type" description:"statsd format, can be either 'standard' or 'datadog' (default 'standard')" env:"GOIARDI_STATSD_TYPE"`
	StatsdInstance    string       `long:"statsd-instance" description:"Statsd instance name to use for this server. Defaults to the server's hostname, with '.' replaced by '_'." env:"GOIARDI_STATSD_INSTANCE"`
	UseS3Upload       bool         `long:"use-s3-upload" description:"Store cookbook files in S3 rather than locally in memory or on disk. This or --local-filestore-dir must be set in SQL mode. Cannot be used with in-memory mode." env:"GOIARDI_USE_S3_UPLOAD"`
	AWSRegion         string       `long:"aws-region" description:"AWS region to use S3 uploads." env:"GOIARDI_AWS_REGION"`
	S3Bucket          string       `long:"s3-bucket" description:"The name of the S3 bucket storing the files." env:"GOIARDI_S3_BUCKET"`
	AWSDisableSSL     bool         `long:"aws-disable-ssl" description:"Set to disable SSL for the endpoint. Mostly useful just for testing." env:"GOIARDI_AWS_DISABLE_SSL"`
	S3Endpoint        string       `long:"s3-endpoint" description:"Set a different endpoint than the default s3.amazonaws.com. Mostly useful for testing with a fake S3 service, or if using an S3-compatible service." env:"GOIARDI_S3_ENDPOINT"`
	S3FilePeriod      int          `long:"s3-file-period" description:"Length of time, in minutes, to allow files to be saved to or retrieved from S3 by the client. Defaults to 15 minutes." env:"GOIARDI_S3_FILE_PERIOD"`
	UseExtSecrets     bool         `long:"use-external-secrets" description:"Use an external service to store secrets (currently user/client public keys). Currently only vault is supported." env:"GOIARDI_USE_EXTERNAL_SECRETS"`
	VaultAddr         string       `long:"vault-addr" description:"Specify address of vault server (i.e. https://127.0.0.1:8200). Defaults to the value of VAULT_ADDR."`
	VaultShoveyKey    string       `long:"vault-shovey-key" description:"Specify a path in vault holding shovey's private key. The key must be put in vault as 'privateKey=<contents>'." env:"GOIARDI_VAULT_SHOVEY_KEY"`
	IndexValTrim      int          `short:"T" long:"index-val-trim" description:"Trim values indexed for chef search to this many characters (keys are untouched). If not set or set <= 0, trimming is disabled. This behavior will change with the next major release." env:"GOIARDI_INDEX_VAL_TRIM"`
	// hidden argument to print a formatted man page to stdout and exit
	PrintManPage bool `long:"print-man-page" hidden:"true"`
}

// The goiardi version.
const Version = "0.11.3"

// The chef version we're at least aiming for, even if it's not complete yet.
const ChefVersion = "11.1.7"

// The default time difference allowed between the server's clock and the time
// in the X-OPS-TIMESTAMP header.
const DefaultTimeSlew = "15m"

/* The general plan is to read the command-line options, then parse the config
 * file, fill in the config struct with those values, then apply the
 * command-line options to the config struct. We read the cli options first so
 * we know to look for a different config file if needed, but otherwise the
 * command line options override what's in the config file. */

func initConfig() *Conf { return &Conf{} }

// Config struct with the options specified on the command line or in the config
// file.
var Config = initConfig()

// ParseConfigOptions reads and applies arguments from the command line and the
// configuration file, merging them together as needed, with command line options
// taking precedence over options in the config file.
func ParseConfigOptions() error {
	var opts = &Options{}
	parser := flags.NewParser(opts, flags.Default)
	parser.ShortDescription = fmt.Sprintf("A Chef server, in Go - version %s", Version)
	parser.LongDescription = "With no arguments, goiardi runs without any authentication or persistence entirely in memory. For authentication, persistence, stability, or other features, run goiardi with the appropriate combination of flags (or set options in the configuration file).\n\nMany of goiardi's command line arguments can be set with environment variables instead of flags, if desired. The options that allow this are followed by the name of the appropriate environment variable (e.g. [$GOIARDI_SOME_OPTION])."
	parser.NamespaceDelimiter = "-"
	if hideVaultOptions {
		vopts := []string{"vault-addr", "vault-shovey-key", "use-external-secrets"}
		for _, v := range vopts {
			c := parser.FindOptionByLongName(v)
			c.Hidden = true
		}
	}
	_, err := parser.Parse()

	if err != nil {
		if err.(*flags.Error).Type == flags.ErrHelp {
			os.Exit(0)
		} else {
			log.Println(err)
			os.Exit(1)
		}
	}
	if opts.PrintManPage {
		parser.LongDescription = strings.Replace(parser.LongDescription, "\n\n", "\n.PP\n", -1)
		parser.WriteManPage(os.Stdout)
		os.Exit(0)
	}

	if opts.Version {
		fmt.Printf("goiardi version %s (git hash: %s) built with %s (aiming for compatibility with Chef Server version %s).\n", Version, GitHash, runtime.Version(), ChefVersion)
		os.Exit(0)
	}

	/* Load the config file. Command-line options have precedence over
	 * config file options. */
	if opts.ConfFile != "" {
		if _, err := toml.DecodeFile(opts.ConfFile, Config); err != nil {
			log.Println(err)
			os.Exit(1)
		}
		Config.ConfFile = opts.ConfFile
		Config.FreezeData = false
	}

	if opts.Export != "" && opts.Import != "" {
		log.Println("Cannot use -x/--export and -m/--import flags together.")
		os.Exit(1)
	}

	if opts.Export != "" {
		Config.DoExport = true
		Config.ImpExFile = opts.Export
	} else if opts.Import != "" {
		Config.DoImport = true
		Config.ImpExFile = opts.Import
	}

	if opts.Hostname != "" {
		Config.Hostname = opts.Hostname
	} else {
		if Config.Hostname == "" {
			Config.Hostname, err = os.Hostname()
			if err != nil {
				log.Println(err)
				Config.Hostname = "localhost"
			}
		}
	}

	if opts.ProxyHostname != "" {
		Config.ProxyHostname = opts.ProxyHostname
	}
	if Config.ProxyHostname == "" {
		Config.ProxyHostname = Config.Hostname
	}

	if opts.DataStoreFile != "" {
		Config.DataStoreFile = opts.DataStoreFile
	}

	if opts.IndexFile != "" {
		Config.IndexFile = opts.IndexFile
	}

	// Use MySQL?
	if opts.UseMySQL {
		Config.UseMySQL = opts.UseMySQL
		// fill in Config with any cli mysql flags
		if opts.MySQL.Username != "" {
			Config.MySQL.Username = opts.MySQL.Username
		}
		if opts.MySQL.Password != "" {
			Config.MySQL.Password = opts.MySQL.Password
		}
		if opts.MySQL.Protocol != "" {
			Config.MySQL.Protocol = opts.MySQL.Protocol
		}
		if opts.MySQL.Address != "" {
			Config.MySQL.Address = opts.MySQL.Address
		}
		if opts.MySQL.Port != "" {
			Config.MySQL.Port = opts.MySQL.Port
		}
		if opts.MySQL.Dbname != "" {
			Config.MySQL.Dbname = opts.MySQL.Dbname
		}
		if opts.MySQL.ExtraParams != nil {
			if Config.MySQL.ExtraParams == nil {
				Config.MySQL.ExtraParams = make(map[string]string)
			}
			for k, v := range opts.MySQL.ExtraParams {
				Config.MySQL.ExtraParams[k] = v
			}
		}
	}

	// Use Postgres?
	if opts.UsePostgreSQL {
		Config.UsePostgreSQL = opts.UsePostgreSQL
		// fill in Config with any cli postgres flags
		if opts.PostgreSQL.Username != "" {
			Config.PostgreSQL.Username = opts.PostgreSQL.Username
		}
		if opts.PostgreSQL.Password != "" {
			Config.PostgreSQL.Password = opts.PostgreSQL.Password
		}
		if opts.PostgreSQL.Host != "" {
			Config.PostgreSQL.Host = opts.PostgreSQL.Host
		}
		if opts.PostgreSQL.Port != "" {
			Config.PostgreSQL.Port = opts.PostgreSQL.Port
		}
		if opts.PostgreSQL.Dbname != "" {
			Config.PostgreSQL.Dbname = opts.PostgreSQL.Dbname
		}
		if opts.PostgreSQL.SSLMode != "" {
			Config.PostgreSQL.SSLMode = opts.PostgreSQL.SSLMode
		}
	}

	if Config.UseMySQL && Config.UsePostgreSQL {
		err := fmt.Errorf("The MySQL and Postgres options cannot be used together.")
		log.Println(err)
		os.Exit(1)
	}

	// Use Postgres search?
	if opts.PgSearch {
		// make sure postgres is enabled
		if !Config.UsePostgreSQL {
			err := fmt.Errorf("--pg-search requires --use-postgresql (which makes sense, really).")
			log.Println(err)
			os.Exit(1)
		}
		Config.PgSearch = opts.PgSearch
	}

	if Config.DataStoreFile != "" && (Config.UseMySQL || Config.UsePostgreSQL) {
		err := fmt.Errorf("The MySQL or Postgres and data store options may not be specified together.")
		log.Println(err)
		os.Exit(1)
	}

	if !((Config.DataStoreFile == "" && Config.IndexFile == "") || ((Config.DataStoreFile != "" || (Config.UseMySQL || Config.UsePostgreSQL)) && Config.IndexFile != "")) {
		err := fmt.Errorf("-i and -D must either both be specified, or not specified")
		log.Println(err)
		os.Exit(1)
	}

	if (Config.UseMySQL || Config.UsePostgreSQL) && (Config.IndexFile == "" && !Config.PgSearch) {
		err := fmt.Errorf("An index file must be specified with -i or --index-file (or the 'index-file' config file option) when running with a MySQL or PostgreSQL backend.")
		log.Println(err)
		os.Exit(1)
	}

	if Config.IndexFile != "" && (Config.DataStoreFile != "" || (Config.UseMySQL || Config.UsePostgreSQL)) {
		Config.FreezeData = true
	}

	if opts.LogFile != "" {
		Config.LogFile = opts.LogFile
	}
	if opts.SysLog {
		Config.SysLog = opts.SysLog
	}
	if Config.LogFile != "" {
		lfp, lerr := os.OpenFile(Config.LogFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, os.ModeAppend|0666)
		if lerr != nil {
			log.Println(err)
			os.Exit(1)
		}
		log.SetOutput(lfp)
	}
	if dlev := len(opts.Verbose); dlev != 0 {
		Config.DebugLevel = dlev
	}
	if opts.LogLevel != "" {
		Config.LogLevel = opts.LogLevel
	}
	if Config.LogLevel != "" {
		if lev, ok := LogLevelNames[strings.ToLower(Config.LogLevel)]; ok && Config.DebugLevel == 0 {
			Config.DebugLevel = lev
		}
	}
	if Config.DebugLevel > 5 {
		Config.DebugLevel = 5
	}

	Config.DebugLevel = int(logger.LevelFatal) - Config.DebugLevel
	logger.SetLevel(logger.LogLevel(Config.DebugLevel))
	debugLevel := map[int]string{0: "debug", 1: "info", 2: "warning", 3: "error", 4: "critical", 5: "fatal"}
	log.Printf("Logging at %s level", debugLevel[Config.DebugLevel])
	// Tired of battling with syslog junk with the logger library. Deal
	// with it ourselves.
	lerr := setLogger(Config.SysLog)
	if lerr != nil {
		log.Println(lerr.Error())
		os.Exit(1)
	}

	/* Database options */

	// Don't bother setting a default mysql port if mysql isn't used
	if Config.UseMySQL {
		if Config.MySQL.Port == "" {
			Config.MySQL.Port = "3306"
		}
	}

	// set default Postgres options
	if Config.UsePostgreSQL {
		if Config.PostgreSQL.Port == "" {
			Config.PostgreSQL.Port = "5432"
		}
	}

	if opts.LocalFstoreDir != "" {
		Config.LocalFstoreDir = opts.LocalFstoreDir
	}

	// s3 upload conf
	if opts.UseS3Upload {
		Config.UseS3Upload = opts.UseS3Upload
	}
	if Config.UseS3Upload {
		if !Config.UseMySQL && !Config.UsePostgreSQL {
			logger.Fatalf("S3 uploads must be used in SQL mode, not in-memory mode.")
			os.Exit(1)
		}
		if opts.AWSRegion != "" {
			Config.AWSRegion = opts.AWSRegion
		}
		if opts.S3Bucket != "" {
			Config.S3Bucket = opts.S3Bucket
		}
		if opts.AWSDisableSSL {
			Config.AWSDisableSSL = opts.AWSDisableSSL
		}
		if opts.S3Endpoint != "" {
			Config.S3Endpoint = opts.S3Endpoint
		}
		if opts.S3FilePeriod != 0 {
			Config.S3FilePeriod = opts.S3FilePeriod
		}

		if Config.S3FilePeriod == 0 {
			Config.S3FilePeriod = 15
		}
	}

	if Config.LocalFstoreDir == "" && ((Config.UseMySQL || Config.UsePostgreSQL) && !Config.UseS3Upload) {
		logger.Fatalf("local-filestore-dir or use-s3-upload must be set and configured when running goiardi in SQL mode")
		os.Exit(1)
	}
	if Config.LocalFstoreDir != "" {
		finfo, ferr := os.Stat(Config.LocalFstoreDir)
		if ferr != nil {
			logger.Fatalf("Error checking local filestore dir: %s", ferr.Error())
			os.Exit(1)
		}
		if !finfo.IsDir() {
			logger.Fatalf("Local filestore dir %s is not a directory", Config.LocalFstoreDir)
			os.Exit(1)
		}
	}

	if !Config.FreezeData && (opts.FreezeInterval != 0 || Config.FreezeInterval != 0) {
		logger.Warningf("FYI, setting the freeze data interval's not especially useful without setting the index and data files.")
	}
	if opts.FreezeInterval != 0 {
		Config.FreezeInterval = opts.FreezeInterval
	}
	if Config.FreezeInterval == 0 {
		Config.FreezeInterval = 10
	}

	/* Root directory for certs and the like */
	if opts.ConfRoot != "" {
		Config.ConfRoot = opts.ConfRoot
	}

	if Config.ConfRoot == "" {
		if Config.ConfFile != "" {
			Config.ConfRoot = path.Dir(Config.ConfFile)
		} else {
			Config.ConfRoot = "."
		}
	}

	if opts.Ipaddress != "" {
		Config.Ipaddress = opts.Ipaddress
	}
	if Config.Ipaddress != "" {
		ip := net.ParseIP(Config.Ipaddress)
		if ip == nil {
			logger.Fatalf("IP address '%s' is not valid", Config.Ipaddress)
			os.Exit(1)
		}
	}

	if opts.Port != 0 {
		Config.Port = opts.Port
	}
	if Config.Port == 0 {
		Config.Port = 4545
	}

	if opts.ProxyPort != 0 {
		Config.ProxyPort = opts.ProxyPort
	}
	if Config.ProxyPort == 0 {
		Config.ProxyPort = Config.Port
	}

	// secret storage config
	if opts.UseExtSecrets {
		Config.UseExtSecrets = opts.UseExtSecrets
	}
	if opts.VaultAddr != "" {
		Config.VaultAddr = opts.VaultAddr
	}

	if opts.UseSSL {
		Config.UseSSL = opts.UseSSL
	}
	if opts.SSLCert != "" {
		Config.SSLCert = opts.SSLCert
	}
	if opts.SSLKey != "" {
		Config.SSLKey = opts.SSLKey
	}
	if opts.HTTPSUrls {
		Config.HTTPSUrls = opts.HTTPSUrls
	}
	// SSL setup
	if Config.Port == 80 {
		Config.UseSSL = false
	} else if Config.Port == 443 {
		Config.UseSSL = true
	}
	if Config.UseSSL {
		if Config.SSLCert == "" || Config.SSLKey == "" {
			logger.Fatalf("SSL mode requires specifying both a certificate and a key file.")
			os.Exit(1)
		}
		/* If the SSL cert and key are not absolute files, join them
		 * with the conf root */
		if !path.IsAbs(Config.SSLCert) {
			Config.SSLCert = path.Join(Config.ConfRoot, Config.SSLCert)
		}
		if !path.IsAbs(Config.SSLKey) {
			Config.SSLKey = path.Join(Config.ConfRoot, Config.SSLKey)
		}
	}

	if opts.TimeSlew != "" {
		Config.TimeSlew = opts.TimeSlew
	}
	if Config.TimeSlew != "" {
		d, derr := time.ParseDuration(Config.TimeSlew)
		if derr != nil {
			logger.Fatalf("Error parsing time-slew: %s", derr.Error())
			os.Exit(1)
		}
		Config.TimeSlewDur = d
	} else {
		Config.TimeSlewDur, _ = time.ParseDuration(DefaultTimeSlew)
	}

	if opts.UseAuth {
		Config.UseAuth = opts.UseAuth
	}

	if opts.DisableWebUI {
		Config.DisableWebUI = opts.DisableWebUI
	}

	if opts.LogEvents {
		Config.LogEvents = opts.LogEvents
	}

	if opts.LogEventKeep != 0 {
		Config.LogEventKeep = opts.LogEventKeep
	}

	// Set max sizes for objects and json requests.
	if opts.ObjMaxSize != 0 {
		Config.ObjMaxSize = opts.ObjMaxSize
	}
	if opts.JSONReqMaxSize != 0 {
		Config.JSONReqMaxSize = opts.JSONReqMaxSize
	}
	if Config.ObjMaxSize == 0 {
		Config.ObjMaxSize = 10485760
	}
	if Config.JSONReqMaxSize == 0 {
		Config.JSONReqMaxSize = 1000000
	}

	if opts.UseUnsafeMemStore {
		Config.UseUnsafeMemStore = opts.UseUnsafeMemStore
		logger.Warningf("UseUnsafeMemStore is deprecated, and will be removed in a future version of goiardi.")
	}

	if opts.DbPoolSize != 0 {
		Config.DbPoolSize = opts.DbPoolSize
	}
	if opts.MaxConn != 0 {
		Config.MaxConn = opts.MaxConn
	}
	if !UsingDB() {
		if Config.DbPoolSize != 0 {
			logger.Infof("db-pool-size is set to %d, which is not particularly useful if you are not using one of the SQL databases.", Config.DbPoolSize)
		}
		if Config.MaxConn != 0 {
			logger.Infof("max-connections is set to %d, which is not particularly useful if you are not using one of the SQL databases.", Config.MaxConn)
		}
	}
	if opts.UseSerf {
		Config.UseSerf = opts.UseSerf
	}
	if Config.UseSerf {
		if opts.SerfAddr != "" {
			Config.SerfAddr = opts.SerfAddr
		}
		if Config.SerfAddr == "" {
			Config.SerfAddr = "127.0.0.1:7373"
		}
	}
	if opts.SerfEventAnnounce {
		Config.SerfEventAnnounce = opts.SerfEventAnnounce
	}
	if Config.SerfEventAnnounce && !Config.UseSerf {
		logger.Fatalf("--serf-event-announce requires --use-serf")
		os.Exit(1)
	}

	if opts.UseShovey {
		if !Config.UseSerf {
			logger.Fatalf("--use-shovey requires --use-serf to be enabled")
			os.Exit(1)
		}
		Config.UseShovey = opts.UseShovey
	}

	// shovey signing key stuff
	if opts.SignPrivKey != "" {
		Config.SignPrivKey = opts.SignPrivKey
	}
	if opts.VaultShoveyKey != "" {
		Config.VaultShoveyKey = opts.VaultShoveyKey
	}

	// if using shovey, open the existing, or create if absent, signing
	// keys.
	if Config.UseShovey {
		if Config.UseExtSecrets {
			if Config.VaultShoveyKey == "" {
				Config.VaultShoveyKey = "keys/shovey/signing"
			}
		} else {
			if Config.SignPrivKey == "" {
				Config.SignPrivKey = path.Join(Config.ConfRoot, "shovey-sign_rsa")
			} else if !path.IsAbs(Config.SignPrivKey) {
				Config.SignPrivKey = path.Join(Config.ConfRoot, Config.SignPrivKey)
			}
			privfp, err := os.Open(Config.SignPrivKey)
			if err != nil {
				logger.Fatalf("Private key %s for signing shovey requests not found. Please create a set of RSA keys for this purpose.", Config.SignPrivKey)
				os.Exit(1)
			}
			privPem, err := ioutil.ReadAll(privfp)
			if err != nil {
				logger.Fatalf(err.Error())
				os.Exit(1)
			}
			privBlock, _ := pem.Decode(privPem)
			if privBlock == nil {
				logger.Fatalf("Invalid block size for private key for shovey")
				os.Exit(1)
			}
			privKey, err := x509.ParsePKCS1PrivateKey(privBlock.Bytes)
			if err != nil {
				logger.Fatalf(err.Error())
				os.Exit(1)
			}
			Key.Lock()
			defer Key.Unlock()
			Key.PrivKey = privKey
		}
	}

	if opts.DotSearch {
		Config.DotSearch = opts.DotSearch
	} else if Config.PgSearch {
		Config.DotSearch = true
	}
	if Config.DotSearch {
		if opts.ConvertSearch {
			Config.ConvertSearch = opts.ConvertSearch
		}
	}
	if Config.IndexFile != "" && Config.PgSearch {
		logger.Infof("Specifying an index file for search while using the postgres search isn't useful.")
	}

	// statsd configuration
	if opts.UseStatsd {
		Config.UseStatsd = opts.UseStatsd
	}
	if opts.StatsdAddr != "" {
		Config.StatsdAddr = opts.StatsdAddr
	}
	if opts.StatsdType != "" {
		Config.StatsdType = opts.StatsdType
	}
	if opts.StatsdInstance != "" {
		Config.StatsdInstance = opts.StatsdInstance
	}
	if Config.StatsdAddr == "" {
		Config.StatsdAddr = "localhost:8125"
	}
	if Config.StatsdType == "" {
		Config.StatsdType = "standard"
	}
	if Config.StatsdInstance == "" {
		Config.StatsdInstance = strings.Replace(Config.Hostname, ".", "_", -1)
	}
	if opts.IndexValTrim != 0 {
		Config.IndexValTrim = opts.IndexValTrim
	}
	if Config.IndexValTrim <= 0 {
		logger.Infof("Trimming values in search index disabled")
		if Config.IndexValTrim == 0 {
			logger.Warningf("index-val-trim's default behavior when not set or set to 0 is to disable search index value trimming; this behavior will change with the next goiardi release")
		}
	} else {
		logger.Infof("Trimming values in search index to %d characters", Config.IndexValTrim)
	}

	// Environment variables
	if len(Config.EnvVars) != 0 {
		for _, v := range Config.EnvVars {
			logger.Debugf("setting %s", v)
			env := strings.SplitN(v, "=", 2)
			if len(env) != 2 {
				logger.Fatalf("Error setting environment variable %s - seems to be malformed.", v)
				os.Exit(1)
			}
			if verr := os.Setenv(env[0], env[1]); verr != nil {
				logger.Fatalf(verr.Error())
				os.Exit(1)
			}
		}
	}

	return nil
}

// ListenAddr builds the address and port goiardi is configured to listen on.
func ListenAddr() string {
	listenAddr := net.JoinHostPort(Config.Ipaddress, strconv.Itoa(Config.Port))
	return listenAddr
}

// ServerHostname returns the hostname and port goiardi is configured to use.
func ServerHostname() string {
	if !(Config.ProxyPort == 80 || Config.ProxyPort == 443) {
		return net.JoinHostPort(Config.ProxyHostname, strconv.Itoa(Config.ProxyPort))
	}
	return Config.ProxyHostname
}

// ServerBaseURL returns the base scheme+hostname portion of a goiardi URL.
func ServerBaseURL() string {
	var urlScheme string
	if Config.UseSSL || Config.HTTPSUrls {
		urlScheme = "https"
	} else {
		urlScheme = "http"
	}
	url := fmt.Sprintf("%s://%s", urlScheme, ServerHostname())
	return url
}

// UsingDB returns true if we're using any db engine, false if using the
// in-memory data store.
func UsingDB() bool {
	return Config.UseMySQL || Config.UsePostgreSQL
}

func UsingExternalSecrets() bool {
	return Config.UseExtSecrets
}
