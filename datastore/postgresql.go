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

// Postgres specific functions for goiardi database work.

package datastore

import (
	"fmt"
	"github.com/ctdk/goiardi/config"
	"strings"
)

func formatPostgresqlConStr(p interface{}) string {
	params := p.(config.PostgreSQLdb)
	var conParams []string

	if params.Username != "" {
		cu := fmt.Sprintf("user=%s", params.Username)
		conParams = append(conParams, cu)
	}
	if params.Password != "" {
		cp := fmt.Sprintf("password=%s", params.Password)
		conParams = append(conParams, cp)
	}
	if params.Host != "" {
		cp := fmt.Sprintf("host=%s", params.Host)
		conParams = append(conParams, cp)
	}
	if params.Port != "" {
		cp := fmt.Sprintf("port=%s", params.Port)
		conParams = append(conParams, cp)
	}
	if params.Dbname != "" {
		cp := fmt.Sprintf("dbname=%s", params.Dbname)
		conParams = append(conParams, cp)
	}
	if params.SSLMode != "" {
		cp := fmt.Sprintf("sslmode=%s", params.SSLMode)
		conParams = append(conParams, cp)
	}

	constr := strings.Join(conParams, " ")

	return constr
}
