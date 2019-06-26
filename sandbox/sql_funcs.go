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

package sandbox

/* Generic SQL functions for sandboxes */

import (
	"database/sql"
	"fmt"
	"github.com/ctdk/goiardi/datastore"
	"github.com/ctdk/goiardi/organization"
	"log"
	"time"
)

func (s *Sandbox) fillSandboxFromSQL(row datastore.ResRow) error {
	return s.fillSandboxFromPostgreSQL(row)
}

func getSQL(org *organization.Organization, sandboxID string) (*Sandbox, error) {
	sandbox := new(Sandbox)
	sandbox.org = org

	sqlStmt := "SELECT sbox_id, creation_time, checksums, completed FROM goiardi.sandboxes WHERE organization_id = $1 AND sbox_id = $2"

	stmt, err := datastore.Dbh.Prepare(sqlStmt)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	row := stmt.QueryRow(org.GetId(), sandboxID)
	err = sandbox.fillSandboxFromSQL(row)
	if err != nil {
		return nil, err
	}
	return sandbox, nil
}

func (s *Sandbox) deleteSQL() error {
	tx, err := datastore.Dbh.Begin()
	if err != nil {
		return err
	}

	sqlStmt := "DELETE FROM goiardi.sandboxes WHERE organization_id = $1 AND sbox_id = $2"

	_, err = tx.Exec(sqlStmt, s.org.GetId(), s.ID)
	if err != nil {
		terr := tx.Rollback()
		if terr != nil {
			err = fmt.Errorf("deleting sandbox %s had an error '%s', and then rolling back the transaction gave another error '%s'", s.ID, err.Error(), terr.Error())
		}
		return err
	}
	tx.Commit()
	return nil
}

func purgeSQL(olderThan time.Time) (int, error) {
	tx, err := datastore.Dbh.Begin()
	if err != nil {
		return 0, err
	}

	sqlStmt := "DELETE FROM goiardi.sandboxes WHERE creation_time < $1"

	res, err := tx.Exec(sqlStmt, olderThan)
	if err != nil {
		terr := tx.Rollback()
		if terr != nil {
			err = fmt.Errorf("deleting sandboxes older than %s had an error '%s', and then rolling back the transaction gave another error '%s'", olderThan.String(), err.Error(), terr.Error())
		}
		return 0, err
	}
	tx.Commit()
	rows, _ := res.RowsAffected()
	return int(rows), nil
}

func getListSQL(org *organization.Organization) []string {
	var sandboxList []string
	sqlStmt := "SELECT sbox_id FROM goiardi.sandboxes WHERE organization_id = $1"
	rows, err := datastore.Dbh.Query(sqlStmt, org.GetId())
	if err != nil {
		if err != sql.ErrNoRows {
			log.Fatal(err)
		}
		rows.Close()
		return sandboxList
	}
	for rows.Next() {
		var sandboxID string
		err = rows.Scan(&sandboxID)
		if err != nil {
			log.Fatal(err)
		}
		sandboxList = append(sandboxList, sandboxID)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		log.Fatal(err)
	}
	return sandboxList
}

func allSandboxesSQL(org *organization.Organization) []*Sandbox {
	var sandboxes []*Sandbox
	sqlStmt := "SELECT sbox_id, creation_time, checksums, completed FROM goiardi.sandboxes WHERE organization_id = $1"

	stmt, err := datastore.Dbh.Prepare(sqlStmt)
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()
	rows, qerr := stmt.Query(org.GetId())
	if qerr != nil {
		if qerr == sql.ErrNoRows {
			return sandboxes
		}
		log.Fatal(qerr)
	}
	for rows.Next() {
		sb := new(Sandbox)
		sb.org = org
		err = sb.fillSandboxFromSQL(rows)
		if err != nil {
			log.Fatal(err)
		}
		sandboxes = append(sandboxes, sb)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		log.Fatal(err)
	}
	return sandboxes
}
