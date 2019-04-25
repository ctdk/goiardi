/*
 * Copyright (c) 2013-2019, Jeremy Bingham (<jeremy@goiardi.gl>)
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

package organization

/* Ye olde general SQL funcs for orgs */

import (
	"database/sql"
	// "github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/datastore"
)

func checkForOrgSQL(dbhandle datastore.Dbhandle, name string) (bool, error) {
	_, err := datastore.CheckForOne(datastore.Dbh, "organizations", name)
	if err == nil {
		return true, nil
	}
	if err != sql.ErrNoRows {
		return false, err
	}
	return false, nil
}

func (o *Organization) fillOrgFromSQL(row datastore.ResRow) error {

}

func getOrgSQL(name string) (*Organization, error) {

}

func (o *Organization) deleteSQL() error {

}

func getListSQL() []string {

}

func allOrgsSQL() []*Organization {

}
