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
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/data_store"
	"log"
)

func checkForUserSQL(dbhandle data_store.Dbhandle, name string) (bool, error) {
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

func (u *User) fillUserFromSQL(row data_store.ResRow) error {
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

func getUserSQL(name string) (*User, error) {
	user := new(User)
	var sqlStatement string
	if config.Config.UseMySQL {
		sqlStatement = "select name, displayname, admin, public_key, email, passwd, salt FROM users WHERE name = ?"
	} else if config.Config.UsePostgreSQL {
		sqlStatement = "select name, displayname, admin, public_key, email, passwd, salt FROM goiardi.users WHERE name = $1"
	}
	stmt, err := data_store.Dbh.Prepare(sqlStatement)
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

func (u *User) deleteSQL() error {
	tx, err := data_store.Dbh.Begin()
	if err != nil {
		return err
	}
	if config.Config.UseMySQL {
		_, err = tx.Exec("DELETE FROM users WHERE name = ?", u.Username)
	} else if config.Config.UsePostgreSQL {
		_, err = tx.Exec("DELETE FROM goiardi.users WHERE name = $1", u.Username)
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
		sqlStatement = "SELECT count(*) FROM users WHERE admin = 1"
	} else if config.Config.UsePostgreSQL {
		sqlStatement = "SELECT count(*) FROM goiardi.users WHERE admin = TRUE"
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
	var user_list []string
	var sqlStatement string
	if config.Config.UseMySQL {
		sqlStatement = "SELECT name FROM users"
	} else if config.Config.UsePostgreSQL {
		sqlStatement = "SELECT name FROM goiardi.users"
	}
	rows, err := data_store.Dbh.Query(sqlStatement)
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

func allUsersSQL() []*User {
	users := make([]*User, 0)
	var sqlStatement string
	if config.Config.UseMySQL {
		sqlStatement = "SELECT name, displayname, admin, public_key, email, passwd, salt FROM users"
	} else if config.Config.UsePostgreSQL {
		sqlStatement = "SELECT name, displayname, admin, public_key, email, passwd, salt FROM goiardi.users"
	}

	stmt, err := data_store.Dbh.Prepare(sqlStatement)
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()
	rows, qerr := stmt.Query()
	if qerr != nil {
		if qerr == sql.ErrNoRows {
			return users
		}
		log.Fatal(qerr)
	}
	for rows.Next() {
		us := new(User)
		err = us.fillUserFromSQL(rows)
		if err != nil {
			log.Fatal(err)
		}
		users = append(users, us)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		log.Fatal(err)
	}
	return users
}
