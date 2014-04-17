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
	"log"
	"net/http"
)

func checkForClientMySQL(dbhandle data_store.Dbhandle, name string) (bool, error) {
	_, err := data_store.CheckForOne(dbhandle, "clients", name)
	if err == nil {
		return true, nil
	} else {
		if err != sql.ErrNoRows {
			return false, err
		} else {
			return false, nil
		}
	}
}

func getClientMySQL(name string) (*Client, error) {
	client := new(Client)
	stmt, err := data_store.Dbh.Prepare("select c.name, nodename, validator, admin, o.name, public_key, certificate FROM clients c JOIN organizations o on c.org_id = o.id WHERE c.name = ?")
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	row := stmt.QueryRow(name)
	err = client.fillClientFromSQL(row)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func (c *Client) fillClientFromSQL(row *sql.Row) error {
	err := row.Scan(&c.Name, &c.NodeName, &c.Validator, &c.Admin, &c.Orgname, &c.pubKey, &c.Certificate)
	if err != nil {
		return err
	}
	c.ChefType = "client"
	c.JsonClass = "Chef::ApiClient"
	return nil
}

func (c *Client) saveMySQL() error {
	tx, err := data_store.Dbh.Begin()
	var client_id int32
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
	client_id, err = data_store.CheckForOne(tx, "clients", c.Name)
	if err == nil {
		_, err := tx.Exec("UPDATE clients SET name = ?, nodename = ?, validator = ?, admin = ?, public_key = ?, certificate = ?, updated_at = NOW() WHERE id = ?", c.Name, c.NodeName, c.Validator, c.Admin, c.pubKey, c.Certificate, client_id)
		if err != nil {
			tx.Rollback()
			return err
		}
	} else {
		if err != sql.ErrNoRows {
			tx.Rollback()
			return err
		}
		_, err = tx.Exec("INSERT INTO clients (name, nodename, validator, admin, public_key, certificate, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, NOW(), NOW())", c.Name, c.NodeName, c.Validator, c.Admin, c.pubKey, c.Certificate)
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	tx.Commit()
	return nil
}

func (c *Client) deleteMySQL() error {
	tx, err := data_store.Dbh.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec("DELETE FROM clients WHERE name = ?", c.Name)
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
	found, err := checkForClientMySQL(data_store.Dbh, new_name)
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

func chkInMemUser (name string) error {
	var err error
	ds := data_store.New()
	if _, found := ds.Get("users", name); found {
		err = fmt.Errorf("a user named %s was found that would conflict with this client", name)
	}
	return err
}

func numAdminsMySQL() int {
	var numAdmins int
	stmt, err := data_store.Dbh.Prepare("SELECT count(*) FROM clients WHERE admin = 1")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()
	err = stmt.QueryRow().Scan(&numAdmins)
	if err != nil {
		log.Fatal(err)
	}
	return numAdmins
}

func getListMySQL() []string {
	var client_list []string
	rows, err := data_store.Dbh.Query("SELECT name FROM clients")
	if err != nil {
		if err != sql.ErrNoRows {
			log.Fatal(err)
		}
		rows.Close()
		return client_list
	}
	client_list = make([]string, 0)
	for rows.Next() {
		var client_name string
		err = rows.Scan(&client_name)
		if err != nil {
			log.Fatal(err)
		}
		client_list = append(client_list, client_name)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		log.Fatal(err)
	}
	return client_list
}
