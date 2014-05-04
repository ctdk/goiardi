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
	"github.com/ctdk/goiardi/data_store"
	"github.com/ctdk/goiardi/util"
	"database/sql"
	"fmt"
	"log"
	"net/http"
)

func checkForUserMySQL(dbhandle data_store.Dbhandle, name string) (bool, error) {
	_, err := data_store.CheckForOne(dbhandle, "users", name)
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

func getUserMySQL(name string) (*User, error) {
	user := new(User)
	stmt, err := data_store.Dbh.Prepare("select name, displayname, admin, public_key, email, passwd, salt FROM users WHERE name = ?")
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

func (u *User) fillUserFromSQL(row *sql.Row) error {
	var email sql.NullString
	err := row.Scan(&u.Username, &u.Name, &u.Admin, &u.pubKey, &email, &u.passwd, &u.salt)
	if err != nil {
		return err
	}
	if !email.Valid {
		u.Email = ""
	} else {
		u.Email = email.String
	}
	return nil
}

func (u *User) saveMySQL() util.Gerror {
	tx, err := data_store.Dbh.Begin()
	var user_id int32
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
	user_id, err = data_store.CheckForOne(tx, "users", u.Username)
	if err == nil {
		_, err := tx.Exec("UPDATE users SET name = ?, displayname = ?, admin = ?, public_key = ?, passwd = ?, salt = ?, updated_at = NOW() WHERE id = ?", u.Username, u.Name, u.Admin, u.pubKey, u.passwd, u.salt, user_id)
		if err != nil {
			tx.Rollback()
			gerr := util.Errorf(err.Error())
			return gerr
		}
	} else {
		if err != sql.ErrNoRows {
			tx.Rollback()
			gerr := util.Errorf(err.Error())
			return gerr
		}
		_, err = tx.Exec("INSERT INTO users (name, displayname, admin, public_key, passwd, salt, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, NOW(), NOW())", u.Username, u.Name, u.Admin, u.pubKey, u.passwd, u.salt)
		if err != nil {
			tx.Rollback()
			gerr := util.Errorf(err.Error())
			return gerr
		}
	}
	tx.Commit()
	return nil
}

func (u *User) deleteMySQL() error {
	tx, err := data_store.Dbh.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec("DELETE FROM users WHERE name = ?", u.Username)
	if err != nil {
		tx.Rollback()
		return err
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
	found, err := checkForUserMySQL(data_store.Dbh, new_name)
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

func chkInMemClient (name string) error {
	var err error
	ds := data_store.New()
	if _, found := ds.Get("clients", name); found {
		err = fmt.Errorf("a client named %s was found that would conflict with this user", name)
	}
	return err
}

func numAdminsMySQL() int {
	var numAdmins int
	stmt, err := data_store.Dbh.Prepare("SELECT count(*) FROM users WHERE admin = 1")
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
	var user_list []string
	rows, err := data_store.Dbh.Query("SELECT name FROM users")
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
