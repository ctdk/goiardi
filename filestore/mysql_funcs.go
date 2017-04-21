/*
 * Copyright (c) 2013-2017, Jeremy Bingham (<jeremy@goiardi.gl>)
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
	"log"
	"strings"

	"github.com/ctdk/goiardi/datastore"
	"github.com/tideland/golib/logger"
)

func (f *FileStore) saveMySQL() error {
	tx, err := datastore.Dbh.Begin()
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

func deleteHashesMySQL(fileHashes []string) {
	if len(fileHashes) == 0 {
		return // nothing to do
	}
	tx, err := datastore.Dbh.Begin()
	if err != nil {
		log.Fatal(err)
	}
	deleteQuery := "DELETE FROM file_checksums WHERE checksum IN(?" + strings.Repeat(",?", len(fileHashes)-1) + ")"
	delArgs := make([]interface{}, len(fileHashes))
	for i, v := range fileHashes {
		delArgs[i] = v
	}
	_, err = tx.Exec(deleteQuery, delArgs...)
	if err != nil && err != sql.ErrNoRows {
		logger.Debugf("Error %s trying to delete hashes", err.Error())
		tx.Rollback()
		return
	}
	tx.Commit()
	return
}
