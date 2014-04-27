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

import (
	"database/sql"
	"fmt"
	"log"
	"github.com/ctdk/goiardi/data_store"
	"time"
)

func (s *Sandbox)fillSandboxFromSQL(row *sql.Row) error {
	var csb []byte
	var tb []byte
	err := row.Scan(&s.Id, &tb, &csb, &s.Completed)
	if err != nil {
		return err
	}
	err = data_store.DecodeBlob(csb, s.Checksums)
	if err != nil {
		return err
	}
	s.CreationTime, err = time.Parse(data_store.MySQLTimeFormat, string(tb))
	if err != nil {
		return err
	}
	return nil
}

func getMySQL(sandbox_id string) (*Sandbox, error) {
	sandbox := new(Sandbox)
	stmt, err := data_store.Dbh.Prepare("SELECT sbox_id, creation_time, checksums, completed FROM sandboxes WHERE sbox_id = ?")
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	row := stmt.QueryRow(sandbox_id)
	err = sandbox.fillSandboxFromSQL(row)
	if err != nil {
		return nil, err
	}
	return sandbox, nil
}

func (s *Sandbox) saveMySQL() error {
	ckb, ckerr := data_store.EncodeBlob(s.Checksums)
	if ckerr != nil {
		return ckerr
	}
	tx, err := data_store.Dbh.Begin()
	if err != nil {
		return err
	}
	var sbox_id string
	err = tx.QueryRow("SELECT sbox_id FROM sandboxes WHERE sbox_id = ?", s.Id).Scan(&sbox_id)
	if err == nil {
		_, err = tx.Exec("UPDATE sandboxes SET checksums = ?, completed = ? WHERE sbox_id = ?", ckb, s.Completed, s.Id)
			if err != nil {
				tx.Rollback()
				return err
			}
	} else {
		if err != sql.ErrNoRows {
			tx.Rollback()
			return err
		}
		_, err = tx.Exec("INSERT INTO sandboxes (sbox_id, creation_time, checksums, completed) VALUES (?, ?, ?, ?)", s.Id, s.CreationTime.UTC().Format(data_store.MySQLTimeFormat), ckb, s.Completed)
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	tx.Commit()
	return nil
}

func (s *Sandbox) deleteMySQL() error {
	tx, err := data_store.Dbh.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec("DELETE FROM sandboxes WHERE sbox_id = ?", s.Id)
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

func getListMySQL() []string {
	sandbox_list := make([]string, 0)
	rows, err := data_store.Dbh.Query("SELECT sbox_id FROM sandboxes")
	if err != nil {
		if err != sql.ErrNoRows {
			log.Fatal(err)
		}
		rows.Close()
		return sandbox_list
	}
	for rows.Next() {
		var sbox_id string
		err = rows.Scan(&sbox_id)
		if err != nil {
			log.Fatal(err)
		}
		sandbox_list = append(sandbox_list, sbox_id)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		log.Fatal(err)
	}
	return sandbox_list
}
