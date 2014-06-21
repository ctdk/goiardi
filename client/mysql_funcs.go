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

package client

import (
	"github.com/ctdk/goiardi/data_store"
	"github.com/ctdk/goiardi/util"
	"database/sql"
	"fmt"
	"net/http"
)

func (c *Client) saveMySQL() error {
	tx, err := data_store.Dbh.Begin()
	if err != nil {
		return err
	}
	// check for a user with this name first. If orgs are ever
	// implemented, it will only need to check for a user 
	// associated with this organization
	err = chkForUser(tx, c.Name)
	if err != nil {
		return err
	}

	_, err = tx.Exec("INSERT INTO clients (name, nodename, validator, admin, public_key, certificate, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, NOW(), NOW()) ON DUPLICATE KEY UPDATE name = ?, nodename = ?, validator = ?, admin = ?, public_key = ?, certificate = ?, updated_at = NOW()", c.Name, c.NodeName, c.Validator, c.Admin, c.pubKey, c.Certificate, c.Name, c.NodeName, c.Validator, c.Admin, c.pubKey, c.Certificate)
	if err != nil {
		tx.Rollback()
		return err
	}

	tx.Commit()
	return nil
}

func (c *Client) renameMySQL(new_name string) util.Gerror {
	tx, err := data_store.Dbh.Begin()
	if err != nil {
		gerr := util.Errorf(err.Error())
		return gerr
	}
	if err = chkForUser(tx, new_name); err != nil {
		tx.Rollback()
		gerr := util.Errorf(err.Error())
		return gerr
	}
	found, err := checkForClientSQL(data_store.Dbh, new_name)
	if found || err != nil {
		tx.Rollback()
		if found && err == nil {
			gerr := util.Errorf("Client %s already exists, cannot rename %s", new_name, c.Name)
			gerr.SetStatus(http.StatusConflict)
			return gerr
		} else {
			gerr := util.Errorf(err.Error())
			gerr.SetStatus(http.StatusInternalServerError)
			return gerr
		}
	}
	_, err = tx.Exec("UPDATE clients SET name = ? WHERE name = ?", new_name, c.Name)
	if err != nil {
		tx.Rollback()
		gerr := util.Errorf(err.Error())
		gerr.SetStatus(http.StatusInternalServerError)
		return gerr
	}
	tx.Commit()
	return nil
}

func chkForUser(handle data_store.Dbhandle, name string) error {
	var user_id int32
	err := handle.QueryRow("SELECT id FROM users WHERE name = ?", name).Scan(&user_id)
	if err != sql.ErrNoRows {
		if err == nil {
			err = fmt.Errorf("a user with id %d named %s was found that would conflict with this client", user_id, name)
		}
	} else {
		err = nil
	}
	return err 
}
