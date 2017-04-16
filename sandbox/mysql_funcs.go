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

package sandbox

/* MySQL functions for sandboxes */

import (
	"github.com/ctdk/goiardi/datastore"
	"time"
)

func (s *Sandbox) fillSandboxFromMySQL(row datastore.ResRow) error {
	var csb []byte
	var tb []byte
	err := row.Scan(&s.ID, &tb, &csb, &s.Completed)
	if err != nil {
		return err
	}
	err = datastore.DecodeBlob(csb, &s.Checksums)
	if err != nil {
		return err
	}
	s.CreationTime, err = time.Parse(datastore.MySQLTimeFormat, string(tb))
	if err != nil {
		return err
	}
	return nil
}

func (s *Sandbox) saveMySQL() error {
	ckb, ckerr := datastore.EncodeBlob(&s.Checksums)
	if ckerr != nil {
		return ckerr
	}
	tx, err := datastore.Dbh.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec("INSERT INTO sandboxes (sbox_id, creation_time, checksums, completed) VALUES (?, ?, ?, ?) ON DUPLICATE KEY UPDATE checksums = ?, completed = ?", s.ID, s.CreationTime.UTC().Format(datastore.MySQLTimeFormat), ckb, s.Completed, ckb, s.Completed)
	if err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()
	return nil
}
