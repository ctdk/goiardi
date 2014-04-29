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

import (
	"github.com/ctdk/goiardi/data_store"
	"fmt"
	"log"
	"database/sql"
)

func checkForRoleMySQL(dbhandle data_store.Dbhandle, name string) (bool, error) {
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

func (r *Role)fillRoleFromSQL(row *sql.Row) error {
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

func getMySQL(role_name string) (*Role, error) {
	role := new(Role)
	stmt, err := data_store.Dbh.Prepare("SELECT name, description, run_list, env_run_lists, default_attr, override_attr FROM roles WHERE name = ?")
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

func (r *Role)saveMySQL() error {
	rlb, rlerr := data_store.EncodeBlob(&r.RunList)
	if rlerr != nil {
		return rlerr
	}
	erb, ererr := data_store.EncodeBlob(&r.EnvRunLists)
	if ererr != nil {
		return ererr
	}
	dab, daerr := data_store.EncodeBlob(&r.Default)
	if daerr != nil {
		return daerr
	}
	oab, oaerr := data_store.EncodeBlob(&r.Override)
	if oaerr != nil {
		return oaerr
	}
	tx, err := data_store.Dbh.Begin()
	var role_id int32
	if err != nil {
		return nil
	}
	role_id, err = data_store.CheckForOne(tx, "roles", r.Name)
	if err == nil {
		_, err := tx.Exec("UPDATE roles SET description = ?, run_list = ?, env_run_lists = ?, default_attr = ?, override_attr = ?, updated_at = NOW() WHERE id = ?", r.Description, rlb, erb, dab, oab, role_id)
		if err != nil {
			tx.Rollback()
			return err
		}
	} else {
		if err != sql.ErrNoRows {
			tx.Rollback()
			return err
		}
		_, err = tx.Exec("INSERT INTO roles (name, description, run_list, env_run_lists, default_attr, override_attr, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, NOW(), NOW())", r.Name, r.Description, rlb, erb, dab, oab)
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	tx.Commit()
	return nil
}

func (r *Role) deleteMySQL() error {
	tx, err := data_store.Dbh.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec("DELETE FROM roles WHERE name = ?", r.Name)
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

func getListMySQL() []string {
	role_list := make([]string, 0)
	rows, err := data_store.Dbh.Query("SELECT name FROM roles")
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
