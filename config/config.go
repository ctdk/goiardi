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
}

/* Struct for command line options. */
type Options struct {
	Version bool `short:"v" long:"version" description:"Print version info."`
	Verbose []bool `short:"V" long:"verbose" description:"Show verbose debug information. (not implemented)"`
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
}

// The goiardi version.
const Version = "0.4.0"
// The chef version we're at least aiming for, even if it's not complete yet.
const ChefVersion = "11.0.8"

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

	if !((Config.DataStoreFile == "" && Config.IndexFile == "") || (Config.DataStoreFile != "" && Config.IndexFile != "")) {
		err := fmt.Errorf("-i and -D must either both be specified, or not specified.")
		panic(err)
		os.Exit(1)
	}

	if Config.IndexFile != "" && Config.DataStoreFile != "" {
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

	if !Config.FreezeData && (opts.FreezeInterval != 0 || Config.FreezeInterval != 0) {
		log.Printf("FYI, setting the freeze data interval's not especially useful without setting the index and data files.")
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
			log.Println("SSL mode requires specifying both a certificate and a key file.")
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

	Config.DebugLevel = len(opts.Verbose)

	if opts.TimeSlew != "" {
		Config.TimeSlew = opts.TimeSlew
	}
	if Config.TimeSlew != "" {
		d, derr := time.ParseDuration(Config.TimeSlew)
		if derr != nil {
			log.Println("Error parsing time-slew:", derr)
			os.Exit(1)
		}
		Config.TimeSlewDur = d
	} else {
		Config.TimeSlewDur, _ = time.ParseDuration("15m")
	}



	if opts.UseAuth {
		Config.UseAuth = opts.UseAuth
	} 

	return nil
}

// The address and port goiardi is configured to listen on.
func ListenAddr() string {
	listen_addr := fmt.Sprintf("%s:%d", Config.Ipaddress, Config.Port)
	return listen_addr
}

// The hostname and port goiardi is configured to use.
func ServerHostname() string {
	var portStr string
	if !(Config.Port == 80 || Config.Port == 443) {
		portStr = fmt.Sprintf(":%d", Config.Port)
	}
	hostname := fmt.Sprintf("%s%s", Config.Hostname, portStr)
	return hostname
}

// The base URL
func ServerBaseURL() string {
	/* TODO: allow configuring using http vs. https */
	var urlScheme string
	if Config.UseSSL || Config.HttpsUrls {
		urlScheme = "https"
	} else {
		urlScheme = "http"
	}
	url := fmt.Sprintf("%s://%s", urlScheme, ServerHostname())
	return url
}
