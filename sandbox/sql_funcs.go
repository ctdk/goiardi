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

package sandbox

/* Generic SQL functions for sandboxes */

import (
	"database/sql"
	"fmt"
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/datastore"
	"log"
)

func (s *Sandbox) fillSandboxFromSQL(row datastore.ResRow) error {
	if config.Config.UseMySQL {
		return s.fillSandboxFromMySQL(row)
	} else if config.Config.UsePostgreSQL {
		return s.fillSandboxFromPostgreSQL(row)
	}
	return nil
}

func getSQL(sandboxID string) (*Sandbox, error) {
	sandbox := new(Sandbox)
	var sqlStmt string
	if config.Config.UseMySQL {
		sqlStmt = "SELECT sbox_id, creation_time, checksums, completed FROM sandboxes WHERE sbox_id = ?"
	} else if config.Config.UsePostgreSQL {
		sqlStmt = "SELECT sbox_id, creation_time, checksums, completed FROM goiardi.sandboxes WHERE sbox_id = $1"
	}
	stmt, err := datastore.Dbh.Prepare(sqlStmt)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	row := stmt.QueryRow(sandboxID)
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
	var sqlStmt string
	if config.Config.UseMySQL {
		sqlStmt = "DELETE FROM sandboxes WHERE sbox_id = ?"
	} else if config.Config.UsePostgreSQL {
		sqlStmt = "DELETE FROM goiardi.sandboxes WHERE sbox_id = $1"
	}
	_, err = tx.Exec(sqlStmt, s.Id)
	if err != nil {
		terr := tx.Rollback()
		if terr != nil {
			err = fmt.Errorf("deleting sandbox %s had an error '%s', and then rolling back the transaction gave another error '%s'", s.Id, err.Error(), terr.Error())
		}
		return err
	}
	tx.Commit()
	return nil
}

func getListSQL() []string {
	var sandboxList []string
	var sqlStmt string
	if config.Config.UseMySQL {
		sqlStmt = "SELECT sbox_id FROM sandboxes"
	} else if config.Config.UsePostgreSQL {
		sqlStmt = "SELECT sbox_id FROM goiardi.sandboxes"
	}
	rows, err := datastore.Dbh.Query(sqlStmt)
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

func allSandboxesSQL() []*Sandbox {
	var sandboxes []*Sandbox
	var sqlStmt string
	if config.Config.UseMySQL {
		sqlStmt = "SELECT sbox_id, creation_time, checksums, completed FROM sandboxes"
	} else if config.Config.UsePostgreSQL {
		sqlStmt = "SELECT sbox_id, creation_time, checksums, completed FROM goiardi.sandboxes"
	}
	stmt, err := datastore.Dbh.Prepare(sqlStmt)
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()
	rows, qerr := stmt.Query()
	if qerr != nil {
		if qerr == sql.ErrNoRows {
			return sandboxes
		}
		log.Fatal(qerr)
	}
	for rows.Next() {
		sb := new(Sandbox)
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
