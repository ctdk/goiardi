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

package search

import (
	"github.com/ctdk/goiardi/datastore"
	"github.com/ctdk/goiardi/util"
)

type PostgresSearch struct {
}

func (p *PostgresSearch) Search(idx string, q string, rows int, sortOrder string, start int, partialData map[string]interface{}) ([]map[string]interface{}, error) {
	return nil, nil
}

func (p *PostgresSearch) GetEndpoints() []string {
	var endpoints util.StringSlice
	stmt, err := datastore.Dbh.Prepare("SELECT ARRAY_AGG(name) FROM goiardi.search_collections WHERE organization_id = $1")
	if err != nil {
		panic(err)
	}
	defer stmt.Close()
	err = stmt.QueryRow(1).Scan(&endpoints)
	if err != nil {
		panic(err)
	}
	return endpoints
}
