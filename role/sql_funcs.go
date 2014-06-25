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

package role

/* Generic SQL funcs for roles */

import (
	"github.com/ctdk/goiardi/data_store"
	"github.com/ctdk/goiardi/config"
	"fmt"
	"log"
	"database/sql"
)

func checkForRoleSQL(dbhandle data_store.Dbhandle, name string) (bool, error) {
	_, err := data_store.CheckForOne(dbhandle, "roles", name)
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

func (r *Role)fillRoleFromSQL(row data_store.ResRow) error {
	var (
		rl []byte
		er []byte
		da []byte
		oa []byte
	)
	err := row.Scan(&r.Name, &r.Description, &rl, &er, &da, &oa)
	if err != nil {
		return err
	}
	r.ChefType = "role"
	r.JsonClass = "Chef::Role"
	err = data_store.DecodeBlob(rl, &r.RunList)
	if err != nil {
		return err
	}
	err = data_store.DecodeBlob(er, &r.EnvRunLists)
	if err != nil {
		return err
	}
	err = data_store.DecodeBlob(da, &r.Default)
	if err != nil {
		return err
	}
	err = data_store.DecodeBlob(oa, &r.Override)
	if err != nil {
		return err
	}
	data_store.ChkNilArray(r)

	return nil
}

func getSQL(role_name string) (*Role, error) {
	role := new(Role)
	var sqlStmt string
	if config.Config.UseMySQL {
		sqlStmt = "SELECT name, description, run_list, env_run_lists, default_attr, override_attr FROM roles WHERE name = ?"
	} else if config.Config.UsePostgreSQL {
		sqlStmt = "SELECT name, description, run_list, env_run_lists, default_attr, override_attr FROM goiardi.roles WHERE name = $1"
	}
	stmt, err := data_store.Dbh.Prepare(sqlStmt)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	row := stmt.QueryRow(role_name)
	err = role.fillRoleFromSQL(row)
	if err != nil {
		return nil, err
	}
	return role, nil
}

func (r *Role) deleteSQL() error {
	tx, err := data_store.Dbh.Begin()
	if err != nil {
		return err
	}
	var sqlStmt string
	if config.Config.UseMySQL {
		sqlStmt = "DELETE FROM roles WHERE name = ?"
	} else if config.Config.UsePostgreSQL {
		sqlStmt = "DELETE FROM goiardi.roles WHERE name = $1"
	}
	_, err = tx.Exec(sqlStmt, r.Name)
	if err != nil {
		terr := tx.Rollback()
		if terr != nil {
			err = fmt.Errorf("deleting role %s had an error '%s', and then rolling back the transaction gave another error '%s'", r.Name, err.Error(), terr.Error())
		}
		return err
	}
	tx.Commit()
	return nil
}

func getListSQL() []string {
	role_list := make([]string, 0)
	var sqlStmt string
	if config.Config.UseMySQL {
		sqlStmt = "SELECT name FROM roles"
	} else if config.Config.UsePostgreSQL {
		sqlStmt = "SELECT name FROM goiardi.roles"
	}
	rows, err := data_store.Dbh.Query(sqlStmt)
	if err != nil {
		rows.Close()
		if err != sql.ErrNoRows {
			log.Fatal(err)
		}
		return role_list
	}
	for rows.Next() {
		var role_name string
		err = rows.Scan(&role_name)
		if err != nil {
			log.Fatal(err)
		}
		role_list = append(role_list, role_name)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		log.Fatal(err)
	}
	return role_list
}

func allRolesSQL() []*Role {
	roles := make([]*Role, 0)
	var sqlStmt string
	if config.Config.UseMySQL {
		sqlStmt = "SELECT name, description, run_list, env_run_lists, default_attr, override_attr FROM roles"
	} else if config.Config.UsePostgreSQL {
		sqlStmt = "SELECT name, description, run_list, env_run_lists, default_attr, override_attr FROM goiardi.roles"
	}
	stmt, err := data_store.Dbh.Prepare(sqlStmt)
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()
	rows, qerr := stmt.Query()
	if qerr != nil {
		if qerr == sql.ErrNoRows {
			return roles
		}
		log.Fatal(qerr)
	}
	for rows.Next() {
		ro := new(Role)
		err = ro.fillRoleFromSQL(rows)
		if err != nil {
			log.Fatal(err)
		}
		roles = append(roles, ro)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		log.Fatal(err)
	}
	return roles
}
