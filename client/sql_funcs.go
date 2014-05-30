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
	"database/sql"
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

func (c *Client) fillClientFromSQL(row *sql.Row) error {
	err := row.Scan(&c.Name, &c.NodeName, &c.Validator, &c.Admin, &c.Orgname, &c.pubKey, &c.Certificate)
	if err != nil {
		return err
	}
	c.ChefType = "client"
	c.JsonClass = "Chef::ApiClient"
	return nil
}
