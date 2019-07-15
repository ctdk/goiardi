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
	"github.com/ctdk/goiardi/datastore"
	"github.com/ctdk/goiardi/util"
	"net/http"
	"strings"
)

func (c *Client) savePostgreSQL() util.Gerror {
	tx, err := datastore.Dbh.Begin()
	if err != nil {
		gerr := util.CastErr(err)
		return gerr
	}
	err = tx.QueryRow("SELECT goiardi.merge_clients($1, $2, $3, $4, $5, $6, $7)", c.Name, c.NodeName, c.Validator, c.Admin, c.pubKey, c.Certificate, c.org.GetId()).Scan(&c.id)
	if err != nil {
		tx.Rollback()
		gerr := util.CastErr(err)
		if strings.HasPrefix(err.Error(), "a user with") {
			gerr.SetStatus(http.StatusConflict)
		}
		return gerr
	}
	tx.Commit()
	return nil
}

func (c *Client) renamePostgreSQL(newName string) util.Gerror {
	tx, err := datastore.Dbh.Begin()
	if err != nil {
		gerr := util.Errorf(err.Error())
		return gerr
	}
	_, err = tx.Exec("SELECT goiardi.rename_client($1, $2, $3)", c.Name, newName, c.org.GetId())
	if err != nil {
		tx.Rollback()
		gerr := util.Errorf(err.Error())
		if strings.HasPrefix(err.Error(), "a user with") || strings.Contains(err.Error(), "already exists, cannot rename") {
			gerr.SetStatus(http.StatusConflict)
		} else {
			gerr.SetStatus(http.StatusInternalServerError)
		}
		return gerr
	}
	tx.Commit()
	return nil
}
