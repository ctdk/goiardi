/*
 * Copyright (c) 2013-2026, Jeremy Bingham (<jeremy@goiardi.gl>)
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
	"io/ioutil"
	"net"
	"os"
	"path"
	"testing"
)

func resetConfig() {
	Config = initConfig()
	pprofWhitelist = nil
}

func TestListenAddr(t *testing.T) {
	resetConfig()
	Config.Ipaddress = "127.0.0.1"
	Config.Port = 4545
	if addr := ListenAddr(); addr != "127.0.0.1:4545" {
		t.Errorf("expected 127.0.0.1:4545, got %s", addr)
	}
}

func TestServerHostname(t *testing.T) {
	resetConfig()
	Config.ProxyHostname = "chef.example.com"
	Config.ProxyPort = 443
	if host := ServerHostname(); host != "chef.example.com" {
		t.Errorf("expected chef.example.com, got %s", host)
	}

	Config.ProxyPort = 8080
	if host := ServerHostname(); host != "chef.example.com:8080" {
		t.Errorf("expected chef.example.com:8080, got %s", host)
	}
}

func TestServerBaseURL(t *testing.T) {
	resetConfig()
	Config.ProxyHostname = "chef.example.com"
	Config.ProxyPort = 80
	Config.UseSSL = false
	Config.HTTPSUrls = false
	if u := ServerBaseURL(); u != "http://chef.example.com" {
		t.Errorf("expected http://chef.example.com, got %s", u)
	}
	Config.HTTPSUrls = true
	if u := ServerBaseURL(); u != "https://chef.example.com" {
		t.Errorf("expected https://chef.example.com, got %s", u)
	}
}

func TestUsingDB(t *testing.T) {
	resetConfig()
	if UsingDB() {
		t.Error("expected UsingDB false by default")
	}
	Config.UsePostgreSQL = true
	if !UsingDB() {
		t.Error("expected UsingDB true with postgres")
	}
}

func TestUsingExternalSecrets(t *testing.T) {
	resetConfig()
	if UsingExternalSecrets() {
		t.Error("expected UsingExternalSecrets false by default")
	}
	Config.UseExtSecrets = true
	if !UsingExternalSecrets() {
		t.Error("expected UsingExternalSecrets true")
	}
}

func TestPprofWhitelisted(t *testing.T) {
	resetConfig()
	pprofWhitelist = []net.IP{net.ParseIP("10.0.0.1")}
	if !PprofWhitelisted(net.ParseIP("10.0.0.1")) {
		t.Error("expected 10.0.0.1 to be whitelisted")
	}
	if PprofWhitelisted(net.ParseIP("10.0.0.2")) {
		t.Error("expected 10.0.0.2 not to be whitelisted")
	}
}

func TestParseConfigOptions(t *testing.T) {
	resetConfig()
	tmpDir, err := ioutil.TempDir("", "goiardi-config-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %s", err.Error())
	}
	defer os.RemoveAll(tmpDir)

	confPath := path.Join(tmpDir, "goiardi.conf")
	confContent := `
hostname = "config-test.example.com"
port = 1234
use-auth = true
`
	if err := ioutil.WriteFile(confPath, []byte(confContent), 0644); err != nil {
		t.Fatalf("failed to write config: %s", err.Error())
	}

	os.Args = []string{"goiardi", "-c", confPath}
	if err := ParseConfigOptions(); err != nil {
		t.Fatalf("ParseConfigOptions() failed: %s", err.Error())
	}
	if Config.Hostname != "config-test.example.com" {
		t.Errorf("expected hostname from config, got %s", Config.Hostname)
	}
	if Config.Port != 1234 {
		t.Errorf("expected port 1234, got %d", Config.Port)
	}
	if !Config.UseAuth {
		t.Error("expected use-auth true from config")
	}
}

func TestListenAddrEmptyIP(t *testing.T) {
	resetConfig()
	Config.Port = 4545
	if addr := ListenAddr(); addr != ":4545" {
		t.Errorf("expected :4545, got %s", addr)
	}
}

func TestServerBaseURLWithSSLPort(t *testing.T) {
	resetConfig()
	Config.ProxyHostname = "chef.example.com"
	Config.ProxyPort = 443
	Config.UseSSL = true
	if u := ServerBaseURL(); u != "https://chef.example.com" {
		t.Errorf("expected https://chef.example.com, got %s", u)
	}
}
