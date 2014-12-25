/* Goiardi configuration. */

/*
 * Copyright (c) 2013-2014, Jeremy Bingham (<jbingham@gmail.com>)
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
	"github.com/BurntSushi/toml"
	"github.com/ctdk/goas/v2/logger"
	"github.com/jessevdk/go-flags"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Conf is the master struct for holding configuration options.
type Conf struct {
	Ipaddress         string
	Port              int
	Hostname          string
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
	ObjMaxSize        int64  `toml:"obj-max-size"`
	JSONReqMaxSize    int64  `toml:"json-req-max-size"`
	UseUnsafeMemStore bool   `toml:"use-unsafe-mem-store"`
	DbPoolSize        int    `toml:"db-pool-size"`
	MaxConn           int    `toml:"max-connections"`
	UseSerf           bool   `toml:"use-serf"`
	SerfEventAnnounce bool   `toml:"serf-event-announce"`
	SerfAddr          string `toml:"serf-addr"`
	UseShovey         bool   `toml:"use-shovey"`
	SignPrivKey       string `toml:"sign-priv-key"`
}

// SigningKeys are the public and private keys for signing shovey requests.
type SigningKeys struct {
	sync.RWMutex
	PrivKey *rsa.PrivateKey
}

// Key is the initialized shovey public and private keys.
var Key = &SigningKeys{}

// LogLevelNames give convenient, easier to remember than number name for the
// different levels of logging.
var LogLevelNames = map[string]int{"debug": 4, "info": 3, "warning": 2, "error": 1, "critical": 0}

// MySQLdb holds MySQL connection options.
type MySQLdb struct {
	Username    string
	Password    string
	Protocol    string
	Address     string
	Port        string
	Dbname      string
	ExtraParams map[string]string `toml:"extra_params"`
}

// PostgreSQLdb holds Postgres connection options.
type PostgreSQLdb struct {
	Username string
	Password string
	Host     string
	Port     string
	Dbname   string
	SSLMode  string
}

// Options holds options set from the command line, which are then merged with
// the options in Conf. Configurations from the command line are preferred to
// those set in the config file.
type Options struct {
	Version           bool   `short:"v" long:"version" description:"Print version info."`
	Verbose           []bool `short:"V" long:"verbose" description:"Show verbose debug information. Repeat for more verbosity."`
	ConfFile          string `short:"c" long:"config" description:"Specify a config file to use."`
	Ipaddress         string `short:"I" long:"ipaddress" description:"Listen on a specific IP address."`
	Hostname          string `short:"H" long:"hostname" description:"Hostname to use for this server. Defaults to hostname reported by the kernel."`
	Port              int    `short:"P" long:"port" description:"Port to listen on. If port is set to 443, SSL will be activated. (default: 4545)"`
	IndexFile         string `short:"i" long:"index-file" description:"File to save search index data to."`
	DataStoreFile     string `short:"D" long:"data-file" description:"File to save data store data to."`
	FreezeInterval    int    `short:"F" long:"freeze-interval" description:"Interval in seconds to freeze in-memory data structures to disk if there have been any changes (requires -i/--index-file and -D/--data-file options to be set). (Default 10 seconds.)"`
	LogFile           string `short:"L" long:"log-file" description:"Log to file X"`
	SysLog            bool   `short:"s" long:"syslog" description:"Log to syslog rather than a log file. Incompatible with -L/--log-file."`
	TimeSlew          string `long:"time-slew" description:"Time difference allowed between the server's clock and the time in the X-OPS-TIMESTAMP header. Formatted like 5m, 150s, etc. Defaults to 15m."`
	ConfRoot          string `long:"conf-root" description:"Root directory for configs and certificates. Default: the directory the config file is in, or the current directory if no config file is set."`
	UseAuth           bool   `short:"A" long:"use-auth" description:"Use authentication. Default: false."`
	UseSSL            bool   `long:"use-ssl" description:"Use SSL for connections. If --port is set to 433, this will automatically be turned on. If it is set to 80, it will automatically be turned off. Default: off. Requires --ssl-cert and --ssl-key."`
	SSLCert           string `long:"ssl-cert" description:"SSL certificate file. If a relative path, will be set relative to --conf-root."`
	SSLKey            string `long:"ssl-key" description:"SSL key file. If a relative path, will be set relative to --conf-root."`
	HTTPSUrls         bool   `long:"https-urls" description:"Use 'https://' in URLs to server resources if goiardi is not using SSL for its connections. Useful when goiardi is sitting behind a reverse proxy that uses SSL, but is communicating with the proxy over HTTP."`
	DisableWebUI      bool   `long:"disable-webui" description:"If enabled, disables connections and logins to goiardi over the webui interface."`
	UseMySQL          bool   `long:"use-mysql" description:"Use a MySQL database for data storage. Configure database options in the config file."`
	UsePostgreSQL     bool   `long:"use-postgresql" description:"Use a PostgreSQL database for data storage. Configure database options in the config file."`
	LocalFstoreDir    string `long:"local-filestore-dir" description:"Directory to save uploaded files in. Optional when running in in-memory mode, *mandatory* for SQL mode."`
	LogEvents         bool   `long:"log-events" description:"Log changes to chef objects."`
	LogEventKeep      int    `short:"K" long:"log-event-keep" description:"Number of events to keep in the event log. If set, the event log will be checked periodically and pruned to this number of entries."`
	Export            string `short:"x" long:"export" description:"Export all server data to the given file, exiting afterwards. Should be used with caution. Cannot be used at the same time as -m/--import."`
	Import            string `short:"m" long:"import" description:"Import data from the given file, exiting afterwards. Cannot be used at the same time as -x/--export."`
	ObjMaxSize        int64  `short:"Q" long:"obj-max-size" description:"Maximum object size in bytes for the file store. Default 10485760 bytes (10MB)."`
	JSONReqMaxSize    int64  `short:"j" long:"json-req-max-size" description:"Maximum size for a JSON request from the client. Per chef-pedant, default is 1000000."`
	UseUnsafeMemStore bool   `long:"use-unsafe-mem-store" description:"Use the faster, but less safe, old method of storing data in the in-memory data store with pointers, rather than encoding the data with gob and giving a new copy of the object to each requestor. If this is enabled goiardi will run faster in in-memory mode, but one goroutine could change an object while it's being used by another. Has no effect when using an SQL backend."`
	DbPoolSize        int    `long:"db-pool-size" description:"Number of idle db connections to maintain. Only useful when using one of the SQL backends. Default is 0 - no idle connections retained"`
	MaxConn           int    `long:"max-connections" description:"Maximum number of connections allowed for the database. Only useful when using one of the SQL backends. Default is 0 - unlimited."`
	UseSerf           bool   `long:"use-serf" description:"If set, have goidari use serf to send and receive events and queries from a serf cluster. Required for shovey."`
	SerfEventAnnounce bool   `long:"serf-event-announce" description:"Announce log events and joining the serf cluster over serf, as serf events. Requires --use-serf."`
	SerfAddr          string `long:"serf-addr" description:"IP address and port to use for RPC communication with a serf agent. Defaults to 127.0.0.1:7373."`
	UseShovey         bool   `long:"use-shovey" description:"Enable using shovey for sending jobs to nodes. Requires --use-serf."`
	SignPrivKey       string `long:"sign-priv-key" description:"Path to RSA private key used to sign shovey requests."`
}

// The goiardi version.
const Version = "0.9.0"

// The chef version we're at least aiming for, even if it's not complete yet.
const ChefVersion = "11.1.3"

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
	_, err := flags.Parse(opts)

	if err != nil {
		if err.(*flags.Error).Type == flags.ErrHelp {
			os.Exit(0)
		} else {
			log.Println(err)
			os.Exit(1)
		}
	}

	if opts.Version {
		fmt.Printf("goiardi version %s (aiming for compatibility with Chef Server version %s).\n", Version, ChefVersion)
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

	if opts.DataStoreFile != "" {
		Config.DataStoreFile = opts.DataStoreFile
	}

	if opts.IndexFile != "" {
		Config.IndexFile = opts.IndexFile
	}

	// Use MySQL?
	if opts.UseMySQL {
		Config.UseMySQL = opts.UseMySQL
	}

	// Use Postgres?
	if opts.UsePostgreSQL {
		Config.UsePostgreSQL = opts.UsePostgreSQL
	}

	if Config.UseMySQL && Config.UsePostgreSQL {
		err := fmt.Errorf("The MySQL and Postgres options cannot be used together.")
		log.Println(err)
		os.Exit(1)
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

	if (Config.UseMySQL || Config.UsePostgreSQL) && Config.IndexFile == "" {
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
		lfp, lerr := os.Create(Config.LogFile)
		if lerr != nil {
			log.Println(err)
			os.Exit(1)
		}
		log.SetOutput(lfp)
	}
	if dlev := len(opts.Verbose); dlev != 0 {
		Config.DebugLevel = dlev
	}
	if Config.LogLevel != "" {
		if lev, ok := LogLevelNames[strings.ToLower(Config.LogLevel)]; ok && Config.DebugLevel == 0 {
			Config.DebugLevel = lev
		}
	}
	if Config.DebugLevel > 4 {
		Config.DebugLevel = 4
	}

	Config.DebugLevel = int(logger.LevelCritical) - Config.DebugLevel
	logger.SetLevel(logger.LogLevel(Config.DebugLevel))
	debugLevel := map[int]string{0: "debug", 1: "info", 2: "warning", 3: "error", 4: "critical"}
	log.Printf("Logging at %s level", debugLevel[Config.DebugLevel])
	if Config.SysLog {
		sl, err := logger.NewSysLogger("goiardi")
		if err != nil {
			log.Println(err.Error())
			os.Exit(1)
		}
		logger.SetLogger(sl)
	} else {
		logger.SetLogger(logger.NewGoLogger())
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
	if Config.LocalFstoreDir == "" && (Config.UseMySQL || Config.UsePostgreSQL) {
		logger.Criticalf("local-filestore-dir must be set when running goiardi in SQL mode")
		os.Exit(1)
	}
	if Config.LocalFstoreDir != "" {
		finfo, ferr := os.Stat(Config.LocalFstoreDir)
		if ferr != nil {
			logger.Criticalf("Error checking local filestore dir: %s", ferr.Error())
			os.Exit(1)
		}
		if !finfo.IsDir() {
			logger.Criticalf("Local filestore dir %s is not a directory", Config.LocalFstoreDir)
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

	Config.Ipaddress = opts.Ipaddress
	if Config.Ipaddress != "" {
		ip := net.ParseIP(Config.Ipaddress)
		if ip == nil {
			logger.Criticalf("IP address '%s' is not valid", Config.Ipaddress)
			os.Exit(1)
		}
	}

	if opts.Port != 0 {
		Config.Port = opts.Port
	}
	if Config.Port == 0 {
		Config.Port = 4545
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
			logger.Criticalf("SSL mode requires specifying both a certificate and a key file.")
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
			logger.Criticalf("Error parsing time-slew: %s", derr.Error())
			os.Exit(1)
		}
		Config.TimeSlewDur = d
	} else {
		Config.TimeSlewDur, _ = time.ParseDuration("15m")
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
		logger.Criticalf("--serf-event-announce requires --use-serf")
		os.Exit(1)
	}

	if opts.UseShovey {
		if !Config.UseSerf {
			logger.Criticalf("--use-shovey requires --use-serf to be enabled")
			os.Exit(1)
		}
		Config.UseShovey = opts.UseShovey
	}

	// shovey signing key stuff
	if opts.SignPrivKey != "" {
		Config.SignPrivKey = opts.SignPrivKey
	}

	// if using shovey, open the existing, or create if absent, signing
	// keys.
	if Config.UseShovey {
		if Config.SignPrivKey == "" {
			Config.SignPrivKey = path.Join(Config.ConfRoot, "shovey-sign_rsa")
		} else if !path.IsAbs(Config.SignPrivKey) {
			Config.SignPrivKey = path.Join(Config.ConfRoot, Config.SignPrivKey)
		}
		privfp, err := os.Open(Config.SignPrivKey)
		if err != nil {
			logger.Criticalf("Private key %s for signing shovey requests not found. Please create a set of RSA keys for this purpose.", Config.SignPrivKey)
			os.Exit(1)
		}
		privPem, err := ioutil.ReadAll(privfp)
		if err != nil {
			logger.Criticalf(err.Error())
			os.Exit(1)
		}
		privBlock, _ := pem.Decode(privPem)
		if privBlock == nil {
			logger.Criticalf("Invalid block size for private key for shovey")
			os.Exit(1)
		}
		privKey, err := x509.ParsePKCS1PrivateKey(privBlock.Bytes)
		if err != nil {
			logger.Criticalf(err.Error())
			os.Exit(1)
		}
		Key.Lock()
		defer Key.Unlock()
		Key.PrivKey = privKey
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
	if !(Config.Port == 80 || Config.Port == 443) {
		return net.JoinHostPort(Config.Hostname, strconv.Itoa(Config.Port))
	}
	return Config.Hostname
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
