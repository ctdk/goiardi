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

package client

import (
	"database/sql"
	"fmt"
	"github.com/ctdk/goiardi/datastore"
	"github.com/ctdk/goiardi/util"
	"net/http"
)

func (c *Client) saveMySQL() error {
	tx, err := datastore.Dbh.Begin()
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

func (c *Client) renameMySQL(newName string) util.Gerror {
	tx, err := datastore.Dbh.Begin()
	if err != nil {
		gerr := util.Errorf(err.Error())
		return gerr
	}
	if err = chkForUser(tx, newName); err != nil {
		tx.Rollback()
		gerr := util.Errorf(err.Error())
		return gerr
	}
	found, err := checkForClientSQL(datastore.Dbh, c.org, newName)
	if found || err != nil {
		tx.Rollback()
		if found && err == nil {
			gerr := util.Errorf("Client %s already exists, cannot rename %s", newName, c.Name)
			gerr.SetStatus(http.StatusConflict)
			return gerr
		}
		gerr := util.Errorf(err.Error())
		gerr.SetStatus(http.StatusInternalServerError)
		return gerr
	}
	_, err = tx.Exec("UPDATE clients SET name = ? WHERE name = ?", newName, c.Name)
	if err != nil {
		tx.Rollback()
		gerr := util.Errorf(err.Error())
		gerr.SetStatus(http.StatusInternalServerError)
		return gerr
	}
	tx.Commit()
	return nil
}

func chkForUser(handle datastore.Dbhandle, name string) error {
	var userID int32
	err := handle.QueryRow("SELECT id FROM users WHERE name = ?", name).Scan(&userID)
	if err != sql.ErrNoRows {
		if err == nil {
			err = fmt.Errorf("a user with id %d named %s was found that would conflict with this client", userID, name)
		}
	} else {
		err = nil
	}
	return err
}
