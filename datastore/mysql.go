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

// MySQL specific functions for goiardi database work.

package datastore

import (
	"fmt"
	"github.com/ctdk/goiardi/config"
	"net"
	"net/url"
	"strings"
)

func formatMysqlConStr(p interface{}) (string, error) {
	params := p.(config.MySQLdb)
	var (
		userpass      string
		protocol      string
		address       string
		dbname        string
		extraParamStr string
	)
	var extraParams []string
	if params.Dbname == "" {
		err := fmt.Errorf("no database name specified")
		return "", err
	}
	dbname = params.Dbname
	if params.Username != "" {
		if params.Password != "" {
			userpass = fmt.Sprintf("%s:%s@", params.Username, params.Password)
		} else {
			userpass = fmt.Sprintf("%s@", params.Username)
		}
	}
	// TODO: see if protocol is needed if address is specified?
	protocol = params.Protocol
	if params.Address != "" {
		var addr string
		if !strings.HasPrefix(protocol, "unix") {
			addr = net.JoinHostPort(params.Address, params.Port)
		} else {
			addr = params.Address
		}
		address = fmt.Sprintf("(%s)", addr)
	}
	for k, v := range params.ExtraParams {
		escVal := url.QueryEscape(v)
		extraParams = append(extraParams, fmt.Sprintf("%s=%s", k, escVal))
	}
	if len(extraParams) != 0 {
		extraParamStr = fmt.Sprintf("?%s", strings.Join(extraParams, "&"))
	}
	connStr := fmt.Sprintf("%s%s%s/%s%s", userpass, protocol, address, dbname, extraParamStr)
	return connStr, nil
}
