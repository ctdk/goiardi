/*
 * Copyright (c) 2013-2016, Jeremy Bingham (<jeremy@goiardi.gl>)
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

// Postgres specific functions for users

import (
	"github.com/ctdk/goiardi/datastore"
	"github.com/ctdk/goiardi/util"
	"net/http"
	"strings"
)

var defaultOrgID = 1

func (u *User) savePostgreSQL() util.Gerror {
	tx, err := datastore.Dbh.Begin()
	if err != nil {
		gerr := util.CastErr(err)
		return gerr
	}
	_, err = tx.Exec("SELECT goiardi.merge_users($1, $2, $3, $4, $5, $6, $7, $8)", u.Username, u.Name, u.Email, u.Admin, u.pubKey, u.passwd, u.salt, defaultOrgID)
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

func (u *User) renamePostgreSQL(newName string) util.Gerror {
	tx, err := datastore.Dbh.Begin()
	if err != nil {
		gerr := util.Errorf(err.Error())
		return gerr
	}
	_, err = tx.Exec("SELECT goiardi.rename_user($1, $2, $3)", u.Username, newName, defaultOrgID)
	if err != nil {
		tx.Rollback()
		gerr := util.Errorf(err.Error())
		if strings.HasPrefix(err.Error(), "a client  with") || strings.Contains(err.Error(), "already exists, cannot rename") {
			gerr.SetStatus(http.StatusConflict)
		} else {
			gerr.SetStatus(http.StatusInternalServerError)
		}
		return gerr
	}
	tx.Commit()
	return nil
}
