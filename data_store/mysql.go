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
package data_store

import (
	"fmt"
	"net/url"
	"strings"
)

func formatMysqlConStr(params map[string]string) (string, error) {
	mainParams := map[string]bool{ "username": true, "password": true, "protocol": true, "address": true, "dbname": true }
	var (
		userpass string
		protocol string
		address string
		dbname string
		extraParamStr string
	)
	extraParams := make([]string, 0)
	if d, found := params["dbname"]; !found {
		err := fmt.Errorf("no database name specified")
		return "", err
	} else {
		dbname = d
	}
	if u, found := params["username"]; found {
		if p, f := params["password"]; f {
			userpass = fmt.Sprintf("%s:%s@", u, p)	
		} else {
			userpass = fmt.Sprintf("%s@", u)
		}
	}
	// TODO: see if protocol is needed if address is specified?
	protocol = params["protocol"]
	if a, found := params["address"]; found {
		address = fmt.Sprintf("(%s)", a)
	}
	for k, v := range params {
		if !mainParams[k] {
			escVal := url.QueryEscape(v)
			extraParams = append(extraParams, fmt.Sprintf("%s=%s", k, escVal))
		}
	}
	if len(extraParams) != 0 {
		extraParamStr = fmt.Sprintf("?%s", strings.Join(extraParams, "&"))
	}
	connStr := fmt.Sprintf("%s%s%s/%s%s", userpass, protocol, address, dbname, extraParamStr)
	return connStr, nil
}
