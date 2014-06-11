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

package filestore

import (
	"github.com/ctdk/goiardi/data_store"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"git.tideland.biz/goas/logger"
)

func getMySQL(chksum string) (*FileStore, error) {
	filestore := new(FileStore)
	stmt, err := data_store.Dbh.Prepare("SELECT checksum FROM file_checksums WHERE checksum = ?")
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	err = stmt.QueryRow(chksum).Scan(&filestore.Chksum)
	if err != nil {
		return nil, err
	}
	return filestore, nil
}

func (f *FileStore) saveMySQL() error {
	tx, err := data_store.Dbh.Begin()
	if err != nil {
		return err
	}
	var chksum string
	err = tx.QueryRow("SELECT checksum FROM file_checksums WHERE checksum = ?", f.Chksum).Scan(&chksum)
	if err != nil { // if err is nil we're just updating the file,
			// don't need a new row
		if err != sql.ErrNoRows {
			tx.Rollback()
			return err
		}
		_, err = tx.Exec("INSERT INTO file_checksums (checksum) VALUES (?)", f.Chksum)
		if err != nil {
			tx.Rollback()
			return err
		}
		tx.Commit()
	}
	return nil
}

func (f *FileStore) deleteMySQL() error {
	tx, err := data_store.Dbh.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec("DELETE FROM file_checksums WHERE checksum = ?", f.Chksum)
	if err != nil {
		terr := tx.Rollback()
		if terr != nil {
			err = fmt.Errorf("deleting file %s had an error '%s', and then rolling back the transaction gave another error '%s'", f.Chksum, err.Error(), terr.Error())
		}
		return err
	}
	tx.Commit()
	return nil
}

func getListMySQL() []string {
	file_list := make([]string, 0)
	stmt, perr := data_store.Dbh.Prepare("SELECT checksum FROM file_checksums")
	if perr != nil {
		if perr != sql.ErrNoRows {
			log.Fatal(perr)
		}
		stmt.Close()
		return file_list
	}
	rows, err := stmt.Query()
	for rows.Next() {
		var chksum string
		err = rows.Scan(&chksum)
		if err != nil {
			log.Fatal(err)
		}
		file_list = append(file_list, chksum)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		log.Fatal(err)
	}
	return file_list
}

func deleteHashesMySQL(file_hashes []string) {
	if len(file_hashes) == 0 {
		return // nothing to do
	}
	tx, err := data_store.Dbh.Begin()
	if err != nil {
		log.Fatal(err)
	}
	delete_query := "DELETE FROM file_checksums WHERE checksum IN(?" + strings.Repeat(",?", len(file_hashes) - 1) + ")"
	del_args := make([]interface{}, len(file_hashes))
	for i, v := range file_hashes {
		del_args[i] = v
	}
	_, err = tx.Exec(delete_query, del_args...)
	if err != nil && err != sql.ErrNoRows {
		logger.Debugf("Error %s trying to delete hashes", err.Error())
		tx.Rollback()
		return
	} 
	tx.Commit()
	return 
}

func allFilestoresSQL() []*FileStore {
	filestores := make([]*FileStore, 0)
	stmt, err := data_store.Dbh.Prepare("SELECT checksum FROM file_checksums")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()
	rows, qerr := stmt.Query()
	if qerr != nil {
		if qerr == sql.ErrNoRows {
			return filestores
		}
		log.Fatal(qerr)
	}
	for rows.Next() {
		fl := new(FileStore)
		err = rows.Scan(&fl.Chksum)
		if err != nil {
			log.Fatal(err)
		}
		if err = fl.loadData(); err != nil {
			log.Fatal(err)
		}
		filestores = append(filestores, fl)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		log.Fatal(err)
	}
	return filestores
}
