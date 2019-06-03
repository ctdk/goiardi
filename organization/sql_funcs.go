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
	"errors"
	"fmt"
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/datastore"
	"github.com/ctdk/goiardi/util"
	"strings"
)

func checkForOrgSQL(dbhandle datastore.Dbhandle, name string) (bool, error) {
	var objID int32
	var prepStatement string
	if config.Config.UseMySQL {
		prepStatement = "SELECT id FROM organizations WHERE name = ?"
	} else if config.Config.UsePostgreSQL {
		prepStatement = "SELECT id FROM goiardi.organizations WHERE name = $1"
	}
	stmt, err := dbhandle.Prepare(prepStatement)
	if err != nil {
		return false, err
	}
	defer stmt.Close()

	err = stmt.QueryRow(name).Scan(&objID)

	if err == nil {
		return true, nil
	} else if err != sql.ErrNoRows {
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

func (o *Organization) saveSQL() util.Gerror {
	// pass-through to the postgres one; may add mysql later, but who knows
	return o.savePostgreSQL()
}

func getOrgSQL(name string) (*Organization, error) {
	var sqlStatement string
	org := new(Organization)

	if config.Config.UseMySQL {
		sqlStatement = "SELECT name, description, guid, uuid, id FROM organizations WHERE name = ?"
	} else if config.Config.UsePostgreSQL {
		sqlStatement = "SELECT name, description, guid, uuid, id FROM goiardi.organizations WHERE name = $1"
	}

	stmt, err := datastore.Dbh.Prepare(sqlStatement)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	row := stmt.QueryRow(name);
	if err = org.fillOrgFromSQL(row); err != nil {
		return nil, err
	}
	return org, nil
}

func (o *Organization) deleteSQL() error {
	var sqlStmt string
	if config.Config.UseMySQL {

	} else {
		sqlStmt = "DELETE FROM goiardi.associations WHERE id = $1"
	}

	tx, err := datastore.Dbh.Begin()
	if err != nil {
		return util.CastErr(err)
	}
	_, err = tx.Exec(sqlStmt, o.id)

	if err != nil {
		tx.Rollback()
		return util.CastErr(err)
	}
	tx.Commit()

	return nil
}

func getListSQL() []string {
	var sqlStatement string
	orgList := make([]string, 0)

	if config.Config.UseMySQL {
		sqlStatement = "SELECT name FROM organizations"
	} else if config.Config.UsePostgreSQL {
		sqlStatement = "SELECT name FROM goiardi.organizations"
	}

	stmt, err := datastore.Dbh.Prepare(sqlStatement)
	if err != nil {
		return nil
	}
	defer stmt.Close()

	rows, qerr := stmt.Query()
	if qerr != nil {
		return nil
	}
	for rows.Next() {
		var s string
		err = rows.Scan(&s)
		if err != nil {
			return nil
		}
		orgList = append(orgList, s)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return nil
	}
	return orgList
}

func allOrgsSQL() ([]*Organization, error) {
	var sqlStatement string
	orgs := make([]*Organization, 0)

	if config.Config.UseMySQL {
		sqlStatement = "SELECT name, description, guid, uuid, id FROM organizations"
	} else if config.Config.UsePostgreSQL {
		sqlStatement = "SELECT name, description, guid, uuid, id FROM goiardi.organizations"
	}

	stmt, err := datastore.Dbh.Prepare(sqlStatement)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, qerr := stmt.Query()
	if qerr != nil {
		if qerr == sql.ErrNoRows {
			return orgs, nil
		}
		return nil, qerr
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

func OrgsByIdSQL(ids []int) ([]*Organization, error) {
	if !config.UsingDB() {
		return nil, errors.New("OrgsByIdSQL only works if you're using a database storage backend.")
	}

	var orgs []*Organization
	var sqlStatement string

	bind := make([]string, len(ids))

	// hmrmph, can't pass in []int as []interface{}, of course.
	intfIds := make([]interface{}, len(ids))

	if config.Config.UseMySQL {
		for i, d := range ids {
			bind[i] = "?"
			intfIds[i] = d
		}

		sqlStatement = fmt.Sprintf("SELECT name, description, guid, uuid, id FROM organizations WHERE id IN (%s)", strings.Join(bind, ", "))
	} else if config.Config.UsePostgreSQL {
		for i, d := range ids {
			bind[i] = fmt.Sprintf("$%d", i + 1)
			intfIds[i] = d
		}
		sqlStatement = fmt.Sprintf("SELECT name, description, guid, uuid, id FROM goiardi.organizations WHERE id IN (%s)", strings.Join(bind, ", "))
	}

	stmt, err := datastore.Dbh.Prepare(sqlStatement)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, qerr := stmt.Query(intfIds...)
	if qerr != nil {
		if qerr == sql.ErrNoRows {
			return orgs, nil
		}
		return nil, qerr
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
