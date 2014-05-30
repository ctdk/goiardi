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

// Postgres specific functions for goiardi database work.
package data_store

import (
	"github.com/ctdk/goiardi/config"
	"fmt"
	//"net"
	//"net/url"
	"strings"
	"log"
)

func formatPostgresqlConStr(p interface{}) string {
	params := p.(config.PostgreSQLdb)
	log.Printf("postgres params: %v", params)
	con_params := make([]string, 0)

	if params.Username != "" {
		cu := fmt.Sprintf("user=%s", params.Username)
		con_params = append(con_params, cu)
	}
	if params.Password != "" {
		cp := fmt.Sprintf("password=%s", params.Password)
		con_params = append(con_params, cp)
	}
	if params.Host != "" {
		cp := fmt.Sprintf("host=%s", params.Host)
		con_params = append(con_params, cp)
	}
	if params.Port != "" {
		cp := fmt.Sprintf("port=%s", params.Port)
		con_params = append(con_params, cp)
	}
	if params.Dbname != "" {
		cp := fmt.Sprintf("dbname=%s", params.Dbname)
		con_params = append(con_params, cp)
	}
	if params.SSLMode != "" {
		cp := fmt.Sprintf("sslmode=%s", params.SSLMode)
		con_params = append(con_params, cp)
	}

	constr := strings.Join(con_params, " ")

	return constr
}
