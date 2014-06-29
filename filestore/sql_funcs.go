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

/* General SQL functions for file store */

package filestore

import (
	"database/sql"
	"fmt"
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/data_store"
	"log"
)

func getSQL(chksum string) (*FileStore, error) {
	filestore := new(FileStore)
	var sqlStatement string
	if config.Config.UseMySQL {
		sqlStatement = "SELECT checksum FROM file_checksums WHERE checksum = ?"
	} else if config.Config.UsePostgreSQL {
		sqlStatement = "SELECT checksum FROM goiardi.file_checksums WHERE checksum = $1"
	}
	stmt, err := data_store.Dbh.Prepare(sqlStatement)
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

func (f *FileStore) deleteSQL() error {
	tx, err := data_store.Dbh.Begin()
	if err != nil {
		return err
	}
	var sqlStatement string
	if config.Config.UseMySQL {
		sqlStatement = "DELETE FROM file_checksums WHERE checksum = ?"
	} else if config.Config.UsePostgreSQL {
		sqlStatement = "DELETE FROM goiardi.file_checksums WHERE checksum = $1"
	}

	_, err = tx.Exec(sqlStatement, f.Chksum)
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

func getListSQL() []string {
	var fileList []string
	var sqlStatement string
	if config.Config.UseMySQL {
		sqlStatement = "SELECT checksum FROM file_checksums"
	} else if config.Config.UsePostgreSQL {
		sqlStatement = "SELECT checksum FROM goiardi.file_checksums"
	}

	stmt, perr := data_store.Dbh.Prepare(sqlStatement)
	if perr != nil {
		if perr != sql.ErrNoRows {
			log.Fatal(perr)
		}
		stmt.Close()
		return fileList
	}
	rows, err := stmt.Query()
	for rows.Next() {
		var chksum string
		err = rows.Scan(&chksum)
		if err != nil {
			log.Fatal(err)
		}
		fileList = append(fileList, chksum)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		log.Fatal(err)
	}
	return fileList
}

func allFilestoresSQL() []*FileStore {
	var filestores []*FileStore
	var sqlStatement string
	if config.Config.UseMySQL {
		sqlStatement = "SELECT checksum FROM file_checksums"
	} else if config.Config.UsePostgreSQL {
		sqlStatement = "SELECT checksum FROM goiardi.file_checksums"
	}

	stmt, err := data_store.Dbh.Prepare(sqlStatement)
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
