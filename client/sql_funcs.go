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
	"github.com/ctdk/goiardi/config"
	"database/sql"
	"log"
)

func checkForClientSQL(dbhandle data_store.Dbhandle, name string) (bool, error) {
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

func (c *Client) fillClientFromSQL(row data_store.ResRow) error {
	err := row.Scan(&c.Name, &c.NodeName, &c.Validator, &c.Admin, &c.Orgname, &c.pubKey, &c.Certificate)
	if err != nil {
		return err
	}
	c.ChefType = "client"
	c.JsonClass = "Chef::ApiClient"
	return nil
}

func getClientSQL(name string) (*Client, error) {
	client := new(Client)
	var sqlStatement string
	if config.Config.UseMySQL {
		sqlStatement = "select c.name, nodename, validator, admin, o.name, public_key, certificate FROM clients c JOIN organizations o on c.organization_id = o.id WHERE c.name = ?"
	} else if config.Config.UsePostgreSQL {
		sqlStatement = "select c.name, nodename, validator, admin, o.name, public_key, certificate FROM goiardi.clients c JOIN goiardi.organizations o on c.organization_id = o.id WHERE c.name = $1"
	}
	stmt, err := data_store.Dbh.Prepare(sqlStatement)
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

func (c *Client) deleteSQL() error {
	tx, err := data_store.Dbh.Begin()
	if err != nil {
		return err
	}
	if config.Config.UseMySQL {
		_, err = tx.Exec("DELETE FROM clients WHERE name = ?", c.Name)
	} else if config.Config.UsePostgreSQL {
		_, err = tx.Exec("DELETE FROM goiardi.clients WHERE name = $1", c.Name)
	}
	if err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()
	return nil
}

func numAdminsSQL() int {
	var numAdmins int
	var sqlStatement string
	if config.Config.UseMySQL {
		sqlStatement = "SELECT count(*) FROM clients WHERE admin = 1"
	} else if config.Config.UsePostgreSQL {
		sqlStatement = "SELECT count(*) FROM goiardi.clients WHERE admin = TRUE"
	}
	stmt, err := data_store.Dbh.Prepare(sqlStatement)
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

func getListSQL() []string {
	var client_list []string
	var sqlStatement string 
	if config.Config.UseMySQL {
		sqlStatement = "SELECT name FROM clients"
	} else if config.Config.UsePostgreSQL {
		sqlStatement = "SELECT name FROM goiardi.clients"
	}
	rows, err := data_store.Dbh.Query(sqlStatement)
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

func allClientsSQL() []*Client {
	clients := make([]*Client, 0)
	stmt, err := data_store.Dbh.Prepare("select c.name, nodename, validator, admin, o.name, public_key, certificate FROM clients c JOIN organizations o on c.organization_id = o.id")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()
	rows, qerr := stmt.Query()
	if qerr != nil {
		if qerr == sql.ErrNoRows {
			return clients
		}
		log.Fatal(qerr)
	}
	for rows.Next() {
		cl := new(Client)
		err = cl.fillClientFromSQL(rows)
		if err != nil {
			log.Fatal(err)
		}
		clients = append(clients, cl)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		log.Fatal(err)
	}
	return clients
}
