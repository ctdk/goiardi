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

package environment

import (
	"database/sql"
	"fmt"
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/datastore"
	"github.com/ctdk/goiardi/organization"
	"log"
	"strings"
)

/* General SQL functions for environments */

func checkForEnvironmentSQL(dbhandle datastore.Dbhandle, org *organization.Organization, name string) (bool, error) {
	_, err := datastore.CheckForOne(dbhandle, "environments", org.GetId(), name)
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
	if e.Name == "_default" {
		e.Default = make(map[string]interface{})
		e.Override = make(map[string]interface{})
		e.CookbookVersions = make(map[string]string)
		return nil
	}

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

func getEnvironmentSQL(envName string, org *organization.Organization) (*ChefEnvironment, error) {
	env := new(ChefEnvironment)
	env.org = org

	sqlStatement := "SELECT name, description, default_attr, override_attr, cookbook_vers FROM goiardi.environments WHERE organization_id = $1 AND name = $2"
	stmt, err := datastore.Dbh.Prepare(sqlStatement)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	row := stmt.QueryRow(org.GetId(), envName)
	err = env.fillEnvFromSQL(row)
	if err != nil {
		return nil, err
	}
	return env, nil
}

func getMultiSQL(envNames []string, org *organization.Organization) ([]*ChefEnvironment, error) {
	bind := make([]string, len(envNames))

	for i := range envNames {
		bind[i] = fmt.Sprintf("$%d", i+2)
	}
	sqlStmt := fmt.Sprintf("SELECT name, description, default_attr, override_attr, cookbook_vers FROM goiardi.environments WHERE organization_id = $1 AND name IN (%s)", strings.Join(bind, ", "))

	stmt, err := datastore.Dbh.Prepare(sqlStmt)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	nameArgs := make([]interface{}, len(envNames)+1)
	nameArgs[0] = org.GetId()

	for i, v := range envNames {
		nameArgs[i] = v
	}
	rows, err := stmt.Query(nameArgs...)
	if err != nil {
		return nil, err
	}
	envs := make([]*ChefEnvironment, 0, len(envNames))
	for rows.Next() {
		e := new(ChefEnvironment)
		e.org = org
		err = e.fillEnvFromSQL(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		envs = append(envs, e)
	}

	rows.Close()
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return envs, nil
}

func (e *ChefEnvironment) deleteEnvironmentSQL() error {
	sqlStatement := "DELETE FROM goiardi.environments WHERE organization_id = $1 AND name = $2"
	tx, err := datastore.Dbh.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec(sqlStatement, e.org.GetId(), e.Name)
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

func getEnvironmentList(org *organization.Organization) []string {
	var envList []string
	sqlStatement := "SELECT name FROM goiardi.environments WHERE organization_id = $1"
	rows, err := datastore.Dbh.Query(sqlStatement, org.GetId())
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

func allEnvironmentsSQL(org *organization.Organization) []*ChefEnvironment {
	var environments []*ChefEnvironment
	sqlStatement := "SELECT name, description, default_attr, override_attr, cookbook_vers FROM goiardi.environments WHERE organization_id = $1 AND name <> '_default'"
	stmt, err := datastore.Dbh.Prepare(sqlStatement, org.GetId())
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
		env.org = org
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
