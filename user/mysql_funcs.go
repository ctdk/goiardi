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

package user

import (
	"database/sql"
	"fmt"
	"github.com/ctdk/goiardi/data_store"
	"github.com/ctdk/goiardi/util"
	"net/http"
)

func (u *User) saveMySQL() util.Gerror {
	tx, err := data_store.Dbh.Begin()
	if err != nil {
		gerr := util.Errorf(err.Error())
		return gerr
	}
	// check for a client with this name first. If orgs are ever
	// implemented, it will only need to check for a client
	// in with this organization
	err = chkForClient(tx, u.Username)
	if err != nil {
		gerr := util.Errorf(err.Error())
		gerr.SetStatus(http.StatusConflict)
		return gerr
	}
	_, err = tx.Exec("INSERT INTO users (name, displayname, admin, public_key, passwd, salt, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, NOW(), NOW()) ON DUPLICATE KEY UPDATE name = ?, displayname = ?, admin = ?, public_key = ?, passwd = ?, salt = ?, updated_at = NOW()", u.Username, u.Name, u.Admin, u.pubKey, u.passwd, u.salt, u.Username, u.Name, u.Admin, u.pubKey, u.passwd, u.salt)
	if err != nil {
		tx.Rollback()
		gerr := util.CastErr(err)
		return gerr
	}
	tx.Commit()
	return nil
}

func (u *User) renameMySQL(new_name string) util.Gerror {
	tx, err := data_store.Dbh.Begin()
	if err != nil {
		gerr := util.Errorf(err.Error())
		return gerr
	}
	if err = chkForClient(tx, new_name); err != nil {
		tx.Rollback()
		gerr := util.Errorf(err.Error())
		return gerr
	}
	found, err := checkForUserSQL(data_store.Dbh, new_name)
	if found || err != nil {
		tx.Rollback()
		if found && err == nil {
			gerr := util.Errorf("User %s already exists, cannot rename %s", new_name, u.Username)
			gerr.SetStatus(http.StatusConflict)
			return gerr
		} else {
			gerr := util.Errorf(err.Error())
			gerr.SetStatus(http.StatusInternalServerError)
			return gerr
		}
	}
	_, err = tx.Exec("UPDATE users SET name = ? WHERE name = ?", new_name, u.Username)
	if err != nil {
		tx.Rollback()
		gerr := util.Errorf(err.Error())
		gerr.SetStatus(http.StatusInternalServerError)
		return gerr
	}
	tx.Commit()
	return nil
}

func chkForClient(handle data_store.Dbhandle, name string) error {
	var user_id int32
	err := handle.QueryRow("SELECT id FROM clients WHERE name = ?", name).Scan(&user_id)
	if err != sql.ErrNoRows {
		if err == nil {
			err = fmt.Errorf("a client with id %d named %s was found that would conflict with this user", user_id, name)
		}
	} else {
		err = nil
	}
	return err
}
