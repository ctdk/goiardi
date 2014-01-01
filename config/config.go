/* Goiardi configuration. */

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

package config

import (
	"github.com/jessevdk/go-flags"
	"os"
	"log"
	"fmt"
)

/* Master struct for configuration. */
type Conf struct {
	IPaddress string
	Port int
	Hostname string
	ConfFile string
	DebugLevel int
}

/* Struct for command line options. */
type Options struct {
	Version bool `short:"v" long:"version" description:"Print version info."`
	Verbose []bool `short:"V" long:"verbose" description:"Show verbose debug information. (not implemented)"`
	ConfFile string `short:"c" long:"config" description:"Specify a config file to use. (not implemented)"`
	IPaddress string `short:"I" long:"ipaddress" description:"Listen on a specific IP address."`
	Hostname string `short:"H" long:"hostname" description:"Hostname to use for this server. Defaults to hostname reported by the kernel."`
	Port int `short:"P" long:"port" description:"Port to listen on." default:"4545"`

}

// The goiardi version
const Version = "0.2.0"
// The chef version we're at least aiming for, even if it's not complete yet
const ChefVersion = "11.0.8"

/* The general plan is to read the command-line options, then parse the config
 * file, fill in the config struct with those values, then apply the 
 * command-line options to the config struct. We read the cli options first so
 * we know to look for a different config file if needed, but otherwise the
 * command line options override what's in the config file. */

func InitConfig() *Conf { return &Conf{ } }

var Config = InitConfig()

// Read and apply arguments from the command line.
func ParseConfigOptions() error {
	var opts = &Options{ }
	_, err := flags.Parse(opts)

	if err != nil {
		if err.(*flags.Error).Type == flags.ErrHelp {
			os.Exit(0)
		} else {
			panic(err)
			os.Exit(1)
		}
	}

	if opts.Version {
		fmt.Printf("goiardi version %s (aiming for compatibility with Chef Server version %s).\n", Version, ChefVersion)
		os.Exit(0)
	}

	/* TODO: load config file. Also specify a default in the options. */
	
	if opts.Hostname != "" {
		Config.Hostname = opts.Hostname
	} else {
		Config.Hostname, err = os.Hostname()
		if err != nil {
			log.Println(err)
			Config.Hostname = "localhost"
		}
	}
	Config.IPaddress = opts.IPaddress
	Config.Port = opts.Port
	if Config.Port == 0 {
		Config.Port = 4545
	}
	Config.DebugLevel = len(opts.Verbose)

	return nil
}

// The address and port goiardi is configured to listen on.
func ListenAddr() string {
	listen_addr := fmt.Sprintf("%s:%d", Config.IPaddress, Config.Port)
	return listen_addr
}

// The hostname and port goiardi is configured to use.
func ServerHostname() string {
	hostname := fmt.Sprintf("%s:%d", Config.Hostname, Config.Port)
	return hostname
}

// The base URL
func ServerBaseURL() string {
	/* TODO: allow configuring using http vs. https */
	url := fmt.Sprintf("http://%s", ServerHostname())
	return url
}
