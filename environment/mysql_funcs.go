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
	"net/http"
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
func (e *ChefEnvironment) fillEnvFromSQL(row *sql.Row) error {
	if config.Config.UseMySQL {
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
		var q interface{}
		q, err = data_store.DecodeBlob(da, e.Default)
		if err != nil {
			return err
		}
		e.Default = q.(map[string]interface{})
		q, err = data_store.DecodeBlob(oa, e.Override)
		if err != nil {
			return err
		}
		e.Override = q.(map[string]interface{})
		q, err = data_store.DecodeBlob(cv, e.CookbookVersions)
		if err != nil {
			return err
		}
		e.CookbookVersions = q.(map[string]string)
		data_store.ChkNilArray(e)
	} else {
		err := fmt.Errorf("no database configured, operating in in-memory mode -- fillEnvFromSQL cannot be run")
		return err
	}
	return nil
}

func getEnvironmentMySQL(env_name string) (*ChefEnvironment, error) {
	env = new(ChefEnvironment)
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
