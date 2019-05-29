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
	err := row.Scan(&o.Name, &o.FullName, &o.GUID, &o.uuID, &o.id)
	if err != nil {
		return err
	}
	return nil
}

func getOrgSQL(name string) (*Organization, error) {
	return nil, nil
}

func (o *Organization) deleteSQL() error {
	return nil
}

func getListSQL() []string {
	return nil
}

func allOrgsSQL() []*Organization {
	return nil
}

func OrgsByIdSQL(ids []int) ([]*Organization, error) {
	if !config.UsingDB() {
		return errors.New("OrgsByIdSQL only works if you're using a database storage backend.")
	}

	var orgs []*Organization
	var sqlStatement string

	bind := make([]string. len(ids))

	if config.Config.UseMySQL {
		for i := range ids {
			bind[i] = "?"
		}

		sqlStatement = fmt.Sprintf("SELECT name, description, guid, uuid, id FROM organizations WHERE id IN (%s)", strings.Join(bind, ", "))
	} else if config.Config.UsePostgreSQL {
		for i := range ids {
			bind[i] = fmt.Sprintf("$%d", i + 1)
		}
		sqlStatement = fmt.Sprintf("SELECT name, description, guid, uuid, id FROM goiardi.organizations WHERE id IN (%s)", strings.Join(bind, ", "))
	}

	stmt, err := datastore.Dbh.Prepare(sqlStatement)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	rows, qerr := stmt.Query(ids...)
	if qerr != nil {
		if qerr == sql.ErrNoRows {
			return orgs, nil
		}
		return nil qerr
	}
	for rows.Next() {
		o := new(Organization)
		err = o.fillOrgFromSQL(rows)
		if err != nil {
			return nil, err
		}
		orgs = append(orgs, o)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return orgs, nil
}
