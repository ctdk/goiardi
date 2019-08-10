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

package user

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/datastore"
	"log"
	"strings"
)

func checkForUserSQL(dbhandle datastore.Dbhandle, name string) (bool, error) {
	var objID int32
	var prepStatement string
	if config.Config.UseMySQL {
		prepStatement = "SELECT id FROM users WHERE name = ?"
	} else if config.Config.UsePostgreSQL {
		prepStatement = "SELECT id FROM goiardi.users WHERE name = $1"
	}
	stmt, err := dbhandle.Prepare(prepStatement)
	if err != nil {
		return false, err
	}
	defer stmt.Close()

	err = stmt.QueryRow(name).Scan(&objID)
	if err == nil {
		return true, nil
	}
	if err != sql.ErrNoRows {
		return false, err
	}
	return false, nil
}

func (u *User) fillUserFromSQL(row datastore.ResRow) error {
	var email sql.NullString
	var fName sql.NullString
	var lName sql.NullString
	var authzId sql.NullString

	err := row.Scan(&u.Username, &u.DisplayName, &u.Admin, &u.pubKey, &email, &u.passwd, &u.salt, &u.id, &fName, &lName, &u.Recoveror, &authzId)
	if err != nil {
		return err
	}

	if !email.Valid {
		u.Email = ""
	} else {
		u.Email = email.String
	}

	if !fName.Valid {
		u.FirstName = ""
	} else {
		u.FirstName = fName.String
	}

	if !lName.Valid {
		u.LastName = ""
	} else {
		u.LastName = lName.String
	}

	if !authzId.Valid {
		u.AuthzID = ""
	} else {
		u.AuthzID = authzId.String
	}

	return nil
}

func getUserSQL(name string) (*User, error) {
	user := new(User)
	sqlStatement := "SELECT name, displayname, admin, public_key, email, passwd, salt, id, first_name, last_name, recoveror, authz_id FROM goiardi.users WHERE name = $1"
	stmt, err := datastore.Dbh.Prepare(sqlStatement)
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
	tx, err := datastore.Dbh.Begin()
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

// This is probably obsolete now.
func numAdminsSQL() int {
	var numAdmins int

	sqlStatement := "SELECT count(*) FROM goiardi.users WHERE admin = TRUE"

	stmt, err := datastore.Dbh.Prepare(sqlStatement)
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
	var userList []string
	sqlStatement := "SELECT name FROM goiardi.users"
	rows, err := datastore.Dbh.Query(sqlStatement)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Fatal(err)
		}
		rows.Close()
		return userList
	}
	userList = make([]string, 0)
	for rows.Next() {
		var userName string
		err = rows.Scan(&userName)
		if err != nil {
			log.Fatal(err)
		}
		userList = append(userList, userName)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		log.Fatal(err)
	}
	return userList
}

func allUsersSQL() []*User {
	var users []*User
	sqlStatement := "SELECT name, displayname, admin, public_key, email, passwd, salt, id, first_name, last_name, recoveror, authz_id FROM goiardi.users"

	stmt, err := datastore.Dbh.Prepare(sqlStatement)
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

func UsersByIdSQL(ids []int64) ([]*User, error) {
	if !config.UsingDB() {
		return nil, errors.New("UsersByIdSQL only works if you're using a database storage backend.")
	}

	var users []*User

	bind := make([]string, len(ids))
	intfIds := make([]interface{}, len(ids))

	for i, d := range ids {
		bind[i] = fmt.Sprintf("$%d", i+1)
		intfIds[i] = d
	}
	sqlStatement := fmt.Sprintf("SELECT name, displayname, admin, public_key, email, passwd, salt, id, first_name, last_name, recoveror, authz_id FROM goiardi.users WHERE id IN (%s)", strings.Join(bind, ", "))

	stmt, err := datastore.Dbh.Prepare(sqlStatement)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	rows, qerr := stmt.Query(intfIds...)
	if qerr != nil {
		if qerr == sql.ErrNoRows {
			return users, nil
		}
		return nil, qerr
	}
	for rows.Next() {
		us := new(User)
		err = us.fillUserFromSQL(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, us)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return users, nil
}
