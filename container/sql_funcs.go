/*
 * Copyright (c) 2013-2019, Jeremy Bingham (<jbingham@gmail.com>)
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

package container

// SQL functions and methods for containers.

import (
	"database/sql"
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/datastore"
	"github.com/ctdk/goiardi/organization"
	"github.com/ctdk/goiardi/orgloader"
)

func checkForContainerSQL(dbhandle datastore.Dbhandle, org *organization.Organization, name string) (bool, error) {
	_, err := datastore.CheckForOne(dbhandle, "containers", org.GetId(), name)
	if err == nil {
		return true, nil
	}
	if err != sql.ErrNoRows {
		return false, err
	}
	return false, nil
}

func (c *Container) fillContainerFromSQL(row datastore.ResRow) error {
	var orgId int64

	err := row.Scan(&c.Name, &orgId)
	if err != nil {
		return err
	}

	org, err := orgloader.OrgByIdSQL(orgId)
	if err != nil {
		return err
	}
	c.Org = org

	return nil
}

func getContainerSQL(name string, org *organization.Organization) (*Container, error) {
	var sqlStatement string
	c := new(Container)

	if config.Config.UseMySQL {
		sqlStatement = "SELECT name, organization_id FROM containers WHERE organization_id = ? AND name = ?"
	} else if config.Config.UsePostgreSQL {
		sqlStatement = "SELECT name, organization_id FROM goiardi.containers WHERE organization_id = $1 AND name = $2"
	}

	stmt, err := datastore.Dbh.Prepare(sqlStatement)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	row := stmt.QueryRow(org.GetId(), name)
	if err = c.fillContainerFromSQL(row); err != nil {
		return nil, err
	}
	return c, nil
}

// There doesn't seem to be any sort of case where you would need to update a
// container once it's been created, so here we get to just go an insert.
func (c *Container) saveSQL() error {
	var sqlStmt string

	// Will we keep MySQL? I'm still uncertain.
	if config.Config.UseMySQL {

	} else {
		sqlStmt = "INSERT INTO goiardi.containers (name, organization_id, created_at, updated_at) VALUES ($1, $2, NOW(), NOW())"
	}

	tx, err := datastore.Dbh.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec(sqlStmt, c.Name, c.Org.GetId())

	if err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()

	return nil
}

func (c *Container) deleteSQL() error {
	var sqlStmt string
	if config.Config.UseMySQL {

	} else {
		sqlStmt = "DELETE FROM goiardi.containers WHERE name = $1 AND organization_id = $2"
	}

	tx, err := datastore.Dbh.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec(sqlStmt, c.Name, c.Org.GetId())

	if err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()

	return nil
}

func getListSQL(org *organization.Organization) []string {
	var sqlStatement string
	containerList := make([]string, 0)

	if config.Config.UseMySQL {
		sqlStatement = "SELECT name FROM containers WHERE organization_id = ?"
	} else if config.Config.UsePostgreSQL {
		sqlStatement = "SELECT name FROM goiardi.containers WHERE organization_id = $1"
	}

	stmt, err := datastore.Dbh.Prepare(sqlStatement)
	if err != nil {
		return nil
	}
	defer stmt.Close()

	rows, qerr := stmt.Query(org.GetId())
	if qerr != nil {
		return nil
	}
	for rows.Next() {
		var s string
		err = rows.Scan(&s)
		if err != nil {
			return nil
		}
		containerList = append(containerList, s)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return nil
	}
	return containerList
}
