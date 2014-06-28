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

/* MySQL specific functions for filestore */

package filestore

import (
	"database/sql"
	"github.com/ctdk/goas/v2/logger"
	"github.com/ctdk/goiardi/data_store"
	"log"
	"strings"
)

func (f *FileStore) saveMySQL() error {
	tx, err := data_store.Dbh.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec("INSERT IGNORE INTO file_checksums (checksum) VALUES (?)", f.Chksum)
	if err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()

	return nil
}

func deleteHashesMySQL(file_hashes []string) {
	if len(file_hashes) == 0 {
		return // nothing to do
	}
	tx, err := data_store.Dbh.Begin()
	if err != nil {
		log.Fatal(err)
	}
	delete_query := "DELETE FROM file_checksums WHERE checksum IN(?" + strings.Repeat(",?", len(file_hashes)-1) + ")"
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
