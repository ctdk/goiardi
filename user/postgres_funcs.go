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

// Postgres specific functions for users

import (
	"github.com/ctdk/goiardi/data_store"
	"github.com/ctdk/goiardi/util"
	"database/sql"
	"log"
	"net/http"
	"strings"
)

var defaultOrgId int = 1

func getUserPostgreSQL(name string) (*User, error) {
	user := new(User)
	stmt, err := data_store.Dbh.Prepare("select name, displayname, admin, public_key, email, passwd, salt FROM goiardi.users WHERE name = $1")
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	row := stmt.QueryRow(name)
	err = user.fillUserFromSQL(row)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (u *User) savePostgreSQL() util.Gerror {
	tx, err := data_store.Dbh.Begin()
	if err != nil {
		gerr := util.CastErr(err)
		return gerr
	}
	_, err = tx.Exec("SELECT goiardi.merge_users($1, $2, $3, $4, $5, $6, $7, $8)", u.Username, u.Name, u.Email, u.Admin, u.pubKey, u.passwd, u.salt, defaultOrgId)
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

func (u *User) deletePostgreSQL() error {
	tx, err := data_store.Dbh.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec("DELETE FROM goiardi.users WHERE name = $1", u.Username)
	if err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()
	return nil
}

func (u *User) renamePostgreSQL(new_name string) util.Gerror {
	tx, err := data_store.Dbh.Begin()
	if err != nil {
		gerr := util.Errorf(err.Error())
		return gerr
	}
	_, err = tx.Exec("SELECT goiardi.rename_user($1, $2, $3)", u.Username, new_name, defaultOrgId)
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

func numAdminsPostgreSQL() int {
	var numAdmins int
	stmt, err := data_store.Dbh.Prepare("SELECT count(*) FROM goiardi.users WHERE admin = TRUE")
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

func getListPostgreSQL() []string {
	var user_list []string
	rows, err := data_store.Dbh.Query("SELECT name FROM goiardi.users")
	if err != nil {
		if err != sql.ErrNoRows {
			log.Fatal(err)
		}
		rows.Close()
		return user_list
	}
	user_list = make([]string, 0)
	for rows.Next() {
		var user_name string
		err = rows.Scan(&user_name)
		if err != nil {
			log.Fatal(err)
		}
		user_list = append(user_list, user_name)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		log.Fatal(err)
	}
	return user_list
}
