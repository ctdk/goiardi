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

package environment

import (
	"github.com/ctdk/goiardi/data_store"
	"github.com/ctdk/goiardi/util"
	"database/sql"
	"fmt"
	"log"
)

/* MySQL specific functions for environments */

func checkForEnvironmentMySQL(dbhandle data_store.Dbhandle, name string) (bool, error) {
	_, err := data_store.CheckForOne(dbhandle, "environments", name)
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

// Fill an environment in from a row returned from the SQL server. See the
// equivalent function in node/node.go for more details.
//
// As there, the SQL query that made the row needs to have the same number &
// order of columns as the one in Get(), even if the WHERE clause is different
// or omitted.
func (e *ChefEnvironment) fillEnvFromSQL(row data_store.ResRow) error {
	var (
		da []byte
		oa []byte
		cv []byte
	)
	err := row.Scan(&e.Name, &e.Description, &da, &oa, &cv)
	if err != nil {
		return err
	}
	e.ChefType = "environment"
	e.JsonClass = "Chef::Environment"
	err = data_store.DecodeBlob(da, &e.Default)
	if err != nil {
		return err
	}
	err = data_store.DecodeBlob(oa, &e.Override)
	if err != nil {
		return err
	}
	err = data_store.DecodeBlob(cv, &e.CookbookVersions)
	if err != nil {
		return err
	}
	data_store.ChkNilArray(e)
	return nil
}

func getEnvironmentMySQL(env_name string) (*ChefEnvironment, error) {
	env := new(ChefEnvironment)
	stmt, err := data_store.Dbh.Prepare("SELECT name, description, default_attr, override_attr, cookbook_vers FROM environments WHERE name = ?")
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	row := stmt.QueryRow(env_name)
	err = env.fillEnvFromSQL(row)
	if err != nil {
		return nil, err
	}
	return env, nil
}

func (e *ChefEnvironment) saveEnvironmentMySQL() util.Gerror {
	dab, daerr := data_store.EncodeBlob(&e.Default)
	if daerr != nil {
		return util.CastErr(daerr)
	}
	oab, oaerr := data_store.EncodeBlob(&e.Override)
	if oaerr != nil {
		return util.CastErr(oaerr)
	}
	cvb, cverr := data_store.EncodeBlob(&e.CookbookVersions)
	if cverr != nil {
		return util.CastErr(cverr)
	}
	tx, err := data_store.Dbh.Begin()
	if err != nil {
		return util.CastErr(err)
	}
	var env_id int32
	env_id, err = data_store.CheckForOne(tx, "environments", e.Name)
	if err == nil {
		_, err := tx.Exec("UPDATE environments SET description = ?, default_attr = ?, override_attr = ?, cookbook_vers = ?, updated_at = NOW() WHERE id = ?", e.Description, dab, oab, cvb, env_id)
		if err != nil {
			tx.Rollback()
			return util.CastErr(err)
		}
	} else {
		if err != sql.ErrNoRows {
			tx.Rollback()
			return util.CastErr(err)
		}
		_, err = tx.Exec("INSERT INTO environments (name, description, default_attr, override_attr, cookbook_vers, created_at, updated_at) VALUES (?, ?, ?, ?, ?, NOW(), NOW())", e.Name, e.Description, dab, oab, cvb)
		if err != nil {
			tx.Rollback()
			return util.CastErr(err)
		}
	}
	tx.Commit()
	return nil
}

func (e *ChefEnvironment) deleteEnvironmentMySQL() error {
	tx, err := data_store.Dbh.Begin()
	if err != nil {
		return err
	}
	/* A convenient trigger takes care of nodes that belonged
	 * to this environment, setting them to _default. */
	_, err = tx.Exec("DELETE FROM environments WHERE name = ?", e.Name)
	if err != nil {
		terr := tx.Rollback()
		if terr != nil {
			err = fmt.Errorf("deleting environment %s had an error '%s', and then rolling back the transaction gave another error '%s'", e.Name, err.Error(), terr.Error())
		}
		return err
	}
	tx.Commit()
	return nil
}

func getEnvironmentList() []string {
	env_list := make([]string, 0)
	rows, err := data_store.Dbh.Query("SELECT name FROM environments")
	if err != nil {
		if err != sql.ErrNoRows {
			log.Fatal(err)
		}
		rows.Close()
		return env_list
	}
	for rows.Next() {
		var env_name string
		err = rows.Scan(&env_name)
		if err != nil {
			log.Fatal(err)
		}
		env_list = append(env_list, env_name)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		log.Fatal(err)
	}
	return env_list
}

func allEnvironmentsSQL() []*ChefEnvironment {
	environments := make([]*ChefEnvironment, 0)
	stmt, err := data_store.Dbh.Prepare("SELECT name, description, default_attr, override_attr, cookbook_vers FROM environments")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()
	rows, qerr := stmt.Query()
	if qerr != nil {
		if qerr == sql.ErrNoRows {
			return environments
		}
		log.Fatal(qerr)
	}
	for rows.Next() {
		env := new(ChefEnvironment)
		err = env.fillEnvFromSQL(rows)
		if err != nil {
			log.Fatal(err)
		}
		environments = append(environments, env)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		log.Fatal(err)
	}
	return environments
}
