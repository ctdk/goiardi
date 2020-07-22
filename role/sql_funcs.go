/*
 * Copyright (c) 2013-2017, Jeremy Bingham (<jeremy@goiardi.gl>)
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
	"database/sql"
	"fmt"
	"log"
	"strings"

	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/datastore"
	"github.com/ctdk/goiardi/util"
)

func checkForRoleSQL(dbhandle datastore.Dbhandle, name string) (bool, error) {
	_, err := datastore.CheckForOne(dbhandle, "roles", name)
	if err == nil {
		return true, nil
	}
	if err != sql.ErrNoRows {
		return false, err
	}
	return false, nil
}

func (r *Role) fillRoleFromSQL(row datastore.ResRow) error {
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
	r.JSONClass = "Chef::Role"
	err = datastore.DecodeBlob(rl, &r.RunList)
	if err != nil {
		return err
	}
	err = datastore.DecodeBlob(er, &r.EnvRunLists)
	if err != nil {
		return err
	}
	err = datastore.DecodeBlob(da, &r.Default)
	if err != nil {
		return err
	}
	err = datastore.DecodeBlob(oa, &r.Override)
	if err != nil {
		return err
	}
	datastore.ChkNilArray(r)

	return nil
}

func getSQL(roleName string) (*Role, error) {
	role := new(Role)
	var sqlStmt string
	if config.Config.UseMySQL {
		sqlStmt = "SELECT name, description, run_list, env_run_lists, default_attr, override_attr FROM roles WHERE name = ?"
	} else if config.Config.UsePostgreSQL {
		sqlStmt = "SELECT name, description, run_list, env_run_lists, default_attr, override_attr FROM goiardi.roles WHERE name = $1"
	}
	stmt, err := datastore.Dbh.Prepare(sqlStmt)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	row := stmt.QueryRow(roleName)
	err = role.fillRoleFromSQL(row)
	if err != nil {
		return nil, err
	}
	return role, nil
}

func getMultiSQL(roleNames []string) ([]*Role, error) {
	var sqlStmt string
	bind := make([]string, len(roleNames))

	if config.Config.UseMySQL {
		for i := range roleNames {
			bind[i] = "?"
		}
		sqlStmt = fmt.Sprintf("SELECT name, description, run_list, env_run_lists, default_attr, override_attr FROM roles WHERE name IN (%s)", strings.Join(bind, ", "))
	} else if config.Config.UsePostgreSQL {
		for i := range roleNames {
			bind[i] = fmt.Sprintf("$%d", i+1)
		}
		sqlStmt = fmt.Sprintf("SELECT name, description, run_list, env_run_lists, default_attr, override_attr FROM goiardi.roles WHERE name IN (%s)", strings.Join(bind, ", "))
	}
	stmt, err := datastore.Dbh.Prepare(sqlStmt)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	nameArgs := make([]interface{}, len(roleNames))
	for i, v := range roleNames {
		nameArgs[i] = v
	}
	rows, err := stmt.Query(nameArgs...)
	if err != nil {
		return nil, err
	}
	roles := make([]*Role, 0, len(roleNames))
	for rows.Next() {
		r := new(Role)
		err = r.fillRoleFromSQL(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		roles = append(roles, r)
	}

	rows.Close()
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return roles, nil
}

func (r *Role) deleteSQL() error {
	tx, err := datastore.Dbh.Begin()
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
	var roleList []string
	var sqlStmt string
	if config.Config.UseMySQL {
		sqlStmt = "SELECT name FROM roles"
	} else if config.Config.UsePostgreSQL {
		sqlStmt = "SELECT name FROM goiardi.roles"
	}
	rows, err := datastore.Dbh.Query(sqlStmt)
	if err != nil {
		rows.Close()
		if err != sql.ErrNoRows {
			log.Fatal(err)
		}
		return roleList
	}
	for rows.Next() {
		var roleName string
		err = rows.Scan(&roleName)
		if err != nil {
			log.Fatal(err)
		}
		roleList = append(roleList, roleName)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		log.Fatal(err)
	}
	return roleList
}

func allRolesSQL() []*Role {
	var roles []*Role
	var sqlStmt string
	if config.Config.UseMySQL {
		sqlStmt = "SELECT name, description, run_list, env_run_lists, default_attr, override_attr FROM roles"
	} else if config.Config.UsePostgreSQL {
		sqlStmt = "SELECT name, description, run_list, env_run_lists, default_attr, override_attr FROM goiardi.roles"
	}
	stmt, err := datastore.Dbh.Prepare(sqlStmt)
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

// Count returns a count of all roles on this server.
func Count() int64 {
	if config.UsingDB() {
		c, _ := util.CountSQL("roles")
		return c
	}
	return int64(len(GetList()))
}
