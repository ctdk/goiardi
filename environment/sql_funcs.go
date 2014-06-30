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
	"database/sql"
	"fmt"
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/datastore"
	"log"
)

/* General SQL functions for environments */

func checkForEnvironmentSQL(dbhandle datastore.Dbhandle, name string) (bool, error) {
	_, err := datastore.CheckForOne(dbhandle, "environments", name)
	if err == nil {
		return true, nil
	}
	if err != sql.ErrNoRows {
		return false, err
	}
	return false, nil
}

// Fill an environment in from a row returned from the SQL server. See the
// equivalent function in node/node.go for more details.
//
// As there, the SQL query that made the row needs to have the same number &
// order of columns as the one in Get(), even if the WHERE clause is different
// or omitted.
func (e *ChefEnvironment) fillEnvFromSQL(row datastore.ResRow) error {
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
	e.JSONClass = "Chef::Environment"
	err = datastore.DecodeBlob(da, &e.Default)
	if err != nil {
		return err
	}
	err = datastore.DecodeBlob(oa, &e.Override)
	if err != nil {
		return err
	}
	err = datastore.DecodeBlob(cv, &e.CookbookVersions)
	if err != nil {
		return err
	}
	datastore.ChkNilArray(e)
	return nil
}

func getEnvironmentSQL(envName string) (*ChefEnvironment, error) {
	env := new(ChefEnvironment)
	var sqlStatement string
	if config.Config.UseMySQL {
		sqlStatement = "SELECT name, description, default_attr, override_attr, cookbook_vers FROM environments WHERE name = ?"
	} else if config.Config.UsePostgreSQL {
		sqlStatement = "SELECT name, description, default_attr, override_attr, cookbook_vers FROM goiardi.environments WHERE name = $1"
	}
	stmt, err := datastore.Dbh.Prepare(sqlStatement)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	row := stmt.QueryRow(envName)
	err = env.fillEnvFromSQL(row)
	if err != nil {
		return nil, err
	}
	return env, nil
}

func (e *ChefEnvironment) deleteEnvironmentSQL() error {
	var sqlStatement string
	if config.Config.UseMySQL {
		sqlStatement = "DELETE FROM environments WHERE name = ?"
	} else if config.Config.UsePostgreSQL {
		sqlStatement = "DELETE FROM goiardi.environments WHERE name = $1"
	}
	tx, err := datastore.Dbh.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec(sqlStatement, e.Name)
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
	var envList []string
	var sqlStatement string
	if config.Config.UseMySQL {
		sqlStatement = "SELECT name FROM environments"
	} else if config.Config.UsePostgreSQL {
		sqlStatement = "SELECT name FROM goiardi.environments"
	}
	rows, err := datastore.Dbh.Query(sqlStatement)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Fatal(err)
		}
		rows.Close()
		return envList
	}
	for rows.Next() {
		var envName string
		err = rows.Scan(&envName)
		if err != nil {
			log.Fatal(err)
		}
		envList = append(envList, envName)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		log.Fatal(err)
	}
	return envList
}

func allEnvironmentsSQL() []*ChefEnvironment {
	var environments []*ChefEnvironment
	var sqlStatement string
	if config.Config.UseMySQL {
		sqlStatement = "SELECT name, description, default_attr, override_attr, cookbook_vers FROM environments WHERE name != '_default'"
	} else if config.Config.UsePostgreSQL {
		sqlStatement = "SELECT name, description, default_attr, override_attr, cookbook_vers FROM goiardi.environments WHERE name <> '_default'"
	}
	stmt, err := datastore.Dbh.Prepare(sqlStatement)
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
