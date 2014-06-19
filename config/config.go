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
	"github.com/jessevdk/go-flags"
	"github.com/BurntSushi/toml"
	"os"
	"log"
	"fmt"
	"time"
	"path"
	"github.com/tideland/goas/v2/logger"
	"strings"
	"net"
	"strconv"
)

/* Master struct for configuration. */
type Conf struct {
	Ipaddress string
	Port int
	Hostname string
	ConfFile string `toml:"conf-file"`
	IndexFile string `toml:"index-file"`
	DataStoreFile string `toml:"data-file"`
	DebugLevel int `toml:"debug-level"`
	LogLevel string `toml:"log-level"`
	FreezeInterval int `toml:"freeze-interval"`
	FreezeData bool `toml:"freeze-data"`
	LogFile string `toml:"log-file"`
	UseAuth bool `toml:"use-auth"`
	TimeSlew string `toml:"time-slew"`
	TimeSlewDur time.Duration
	ConfRoot string `toml:"conf-root"`
	UseSSL bool `toml:"use-ssl"`
	SslCert string `toml:"ssl-cert"`
	SslKey string `toml:"ssl-key"`
	HttpsUrls bool `toml:"https-urls"`
	DisableWebUI bool `toml:"disable-webui"`
	UseMySQL bool `toml:"use-mysql"`
	MySQL MySQLdb `toml:"mysql"`
	LocalFstoreDir string `toml:"local-filestore-dir"`
	LogEvents bool `toml:"log-events"`
	LogEventKeep int `toml:"log-event-keep"`
	DoExport bool
	DoImport bool
	ImpExFile string
	ObjMaxSize int64 `toml:"obj-max-size"`
	JsonReqMaxSize int64 `toml:"json-req-max-size"`
}
var LogLevelNames = map[string]int{ "debug": 4, "info": 3, "warning": 2, "error": 1, "critical": 0 }

// MySQL connection options
type MySQLdb struct {
	Username string
	Password string
	Protocol string
	Address string
	Port string
	Dbname string
	ExtraParams map[string]string `toml:"extra_params"`
}

/* Struct for command line options. */
type Options struct {
	Version bool `short:"v" long:"version" description:"Print version info."`
	Verbose []bool `short:"V" long:"verbose" description:"Show verbose debug information. Repeat for more verbosity."`
	ConfFile string `short:"c" long:"config" description:"Specify a config file to use."`
	Ipaddress string `short:"I" long:"ipaddress" description:"Listen on a specific IP address."`
	Hostname string `short:"H" long:"hostname" description:"Hostname to use for this server. Defaults to hostname reported by the kernel."`
	Port int `short:"P" long:"port" description:"Port to listen on. If port is set to 443, SSL will be activated. (default: 4545)"`
	IndexFile string `short:"i" long:"index-file" description:"File to save search index data to."`
	DataStoreFile string `short:"D" long:"data-file" description:"File to save data store data to."`
	FreezeInterval int `short:"F" long:"freeze-interval" description:"Interval in seconds to freeze in-memory data structures to disk (requires -i/--index-file and -D/--data-file options to be set). (Default 300 seconds/5 minutes.)"`
	LogFile string `short:"L" long:"log-file" description:"Log to file X"`
	TimeSlew string `long:"time-slew" description:"Time difference allowed between the server's clock at the time in the X-OPS-TIMESTAMP header. Formatted like 5m, 150s, etc. Defaults to 15m."`
	ConfRoot string `long:"conf-root" description:"Root directory for configs and certificates. Default: the directory the config file is in, or the current directory if no config file is set."`
	UseAuth bool `short:"A" long:"use-auth" description:"Use authentication. Default: false."`
	UseSSL bool `long:"use-ssl" description:"Use SSL for connections. If --port is set to 433, this will automatically be turned on. If it is set to 80, it will automatically be turned off. Default: off. Requires --ssl-cert and --ssl-key."`
	SslCert string `long:"ssl-cert" description:"SSL certificate file. If a relative path, will be set relative to --conf-root."`
	SslKey string `long:"ssl-key" description:"SSL key file. If a relative path, will be set relative to --conf-root."`
	HttpsUrls bool `long:"https-urls" description:"Use 'https://' in URLs to server resources if goiardi is not using SSL for its connections. Useful when goiardi is sitting behind a reverse proxy that uses SSL, but is communicating with the proxy over HTTP."`
	DisableWebUI bool `long:"disable-webui" description:"If enabled, disables connections and logins to goiardi over the webui interface."`
	UseMySQL bool `long:"use-mysql" description:"Use a MySQL database for data storage. Configure database options in the config file."`
	LocalFstoreDir string `long:"local-filestore-dir" description:"Directory to save uploaded files in. Optional when running in in-memory mode, *mandatory* for SQL mode."`
	LogEvents bool `long:"log-events" description:"Log changes to chef objects."`
	LogEventKeep int `short:"K" long:"log-event-keep" description:"Number of events to keep in the event log. If set, the event log will be checked periodically and pruned to this number of entries."`
	Export string `short:"x" long:"export" description:"Export all server data to the given file, exiting afterwards. Should be used with caution. Cannot be used at the same time as -m/--import."`
	Import string `short:"m" long:"import" description:"Import data from the given file, exiting afterwards. Cannot be used at the same time as -x/--export."`
	ObjMaxSize int64 `short:"Q" long:"obj-max-size" description:"Maximum object size in bytes for the file store. Default 10485760 bytes (10MB)."`
	JsonReqMaxSize int64 `short:"j" long:"json-req-max-size" description:"Maximum size for a JSON request from the client. Per chef-pedant, default is 1000000."`
}

// The goiardi version.
const Version = "0.5.2"
// The chef version we're at least aiming for, even if it's not complete yet.
const ChefVersion = "11.0.11"

/* The general plan is to read the command-line options, then parse the config
 * file, fill in the config struct with those values, then apply the 
 * command-line options to the config struct. We read the cli options first so
 * we know to look for a different config file if needed, but otherwise the
 * command line options override what's in the config file. */

func InitConfig() *Conf { return &Conf{ } }

// Conf struct with the options specified on the command line or in the config
// file.
var Config = InitConfig()

// Read and apply arguments from the command line.
func ParseConfigOptions() error {
	var opts = &Options{ }
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
			panic(err)
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

	if Config.DataStoreFile != "" && Config.UseMySQL {
		err := fmt.Errorf("The MySQL and data store options may not be specified together.")
		log.Println(err)
		os.Exit(1)
	}

	if !((Config.DataStoreFile == "" && Config.IndexFile == "") || ((Config.DataStoreFile != "" || Config.UseMySQL) && Config.IndexFile != "")) {
		err := fmt.Errorf("-i and -D must either both be specified, or not specified.")
		log.Println(err)
		os.Exit(1)
	}

	if Config.UseMySQL && Config.IndexFile == "" {
		err := fmt.Errorf("An index file must be specified with -i or --index-file (or the 'index-file' config file option) when running with a MySQL backend.")
		log.Println(err)
		os.Exit(1)
	}

	if Config.IndexFile != "" && (Config.DataStoreFile != "" || Config.UseMySQL) {
		Config.FreezeData = true
	}

	if opts.LogFile != "" {
		Config.LogFile = opts.LogFile
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
	debug_level := map[int]string { 0: "debug", 1: "info", 2: "warning", 3: "error", 4: "critical" }
	log.Printf("Logging at %s level", debug_level[Config.DebugLevel])
	logger.SetLogger(logger.NewGoLogger())

	/* Database options */
	
	// Don't bother setting a default mysql port if mysql isn't used
	if Config.UseMySQL {
		if Config.MySQL.Port == "" {
			Config.MySQL.Port = "3306"
		}
	}

	if opts.LocalFstoreDir != "" {
		Config.LocalFstoreDir = opts.LocalFstoreDir
	}
	if Config.LocalFstoreDir == "" && Config.UseMySQL {
		logger.Criticalf("local-filestore-dir must be set when running goiardi in SQL mode")
		os.Exit(1)
	}

	if !Config.FreezeData && (opts.FreezeInterval != 0 || Config.FreezeInterval != 0) {
		logger.Warningf("FYI, setting the freeze data interval's not especially useful without setting the index and data files.")
	}
	if opts.FreezeInterval != 0 {
		Config.FreezeInterval = opts.FreezeInterval
	}
	if Config.FreezeInterval == 0 {
		Config.FreezeInterval = 300
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
	if opts.Port != 0 {
		Config.Port = opts.Port
	}
	if Config.Port == 0 {
		Config.Port = 4545
	}

	if opts.UseSSL {
		Config.UseSSL = opts.UseSSL
	}
	if opts.SslCert != "" {
		Config.SslCert = opts.SslCert
	}
	if opts.SslKey != "" {
		Config.SslKey = opts.SslKey
	}
	if opts.HttpsUrls {
		Config.HttpsUrls = opts.HttpsUrls
	}
	// SSL setup
	if Config.Port == 80 {
		Config.UseSSL = false
	} else if Config.Port == 443 {
		Config.UseSSL = true
	}
	if Config.UseSSL {
		if Config.SslCert == "" || Config.SslKey == "" {
			logger.Criticalf("SSL mode requires specifying both a certificate and a key file.")
			os.Exit(1)
		}
		/* If the SSL cert and key are not absolute files, join them
		 * with the conf root */
		if !path.IsAbs(Config.SslCert) {
			Config.SslCert = path.Join(Config.ConfRoot, Config.SslCert)
		}
		if !path.IsAbs(Config.SslKey) {
			Config.SslKey = path.Join(Config.ConfRoot, Config.SslKey)
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
	if opts.JsonReqMaxSize != 0 {
		Config.JsonReqMaxSize = opts.JsonReqMaxSize
	}
	if Config.ObjMaxSize == 0 {
		Config.ObjMaxSize = 10485760
	}
	if Config.JsonReqMaxSize == 0 {
		Config.JsonReqMaxSize = 1000000
	}

	return nil
}

// The address and port goiardi is configured to listen on.
func ListenAddr() string {
	listen_addr := net.JoinHostPort(Config.Ipaddress, strconv.Itoa(Config.Port))
	return listen_addr
}

// The hostname and port goiardi is configured to use.
func ServerHostname() string {
	if !(Config.Port == 80 || Config.Port == 443) {
		return net.JoinHostPort(Config.Hostname, strconv.Itoa(Config.Port))
	} else {
		return Config.Hostname
	}
}

// The base URL
func ServerBaseURL() string {
	var urlScheme string
	if Config.UseSSL || Config.HttpsUrls {
		urlScheme = "https"
	} else {
		urlScheme = "http"
	}
	url := fmt.Sprintf("%s://%s", urlScheme, ServerHostname())
	return url
}
