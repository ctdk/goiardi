/*
 * Copyright (c) 2013-2020, Jeremy Bingham (<jeremy@goiardi.gl>)
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

package policies

import (
	"database/sql"
	"github.com/ctdk/goiardi/datastore"
	"github.com/ctdk/goiardi/organization"
	"golang.org/x/xerrors"
)

func checkForPolicySQL(dbhandle datastore.Dbhandle, org *organization.Organization, name string) (bool, error) {
	_, err := datastore.CheckForOne(dbhandle, "policies", org.GetId(), name)
	if err == nil {
		return true, nil
	}

	if !xerrors.Is(err, sql.ErrNoRows) {
		return false, err
	}
	return false, nil
}

func getPolicySQL(org *organization.Organization, name string) (*Policy, error) {
	p := new(Policy)
	p.org = org

	sqlStatement := "SELECT id, name FROM goiardi.policies WHERE organization_id = $1 AND name = $2"
	stmt, err := datastore.Dbh.Prepare(sqlStatement)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	row := stmt.QueryRow(org.GetId(), name)
	err = p.fillPolicyFromSQL(row)
	if err != nil {
		return nil, err
	}

	pRevs, err := p.getAllRevisionsSQL()
	if err != nil {
		return nil, err
	}
	p.Revisions = pRevs

	return p, nil
}

func (p *Policy) fillPolicyFromSQL(row datastore.ResRow) error {
	err := row.Scan(&p.id, &p.Name)
	if err != nil {
		return err
	}
	return nil
}

func (p *Policy) savePolicySQL() error {

	return nil
}

func (p *Policy) deletePolicySQL() error {
	tx, err := datastore.Dbh.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec("DELETE FROM goiardi.policies WHERE id = $1", p.id)
	if err != nil {
		werr := xerrors.Errorf("deleting policy %s had an error: %w", p.Name, err)
		terr := tx.Rollback()
		if terr != nil {
			werr = xerrors.Errorf("%s and then rolling back the transaction gave another error: %w", terr)
		}
		return werr
	}
	tx.Commit()

	return nil
}

// actually needed?
func getPolicyListSQL(org *organization.Organization) ([]string, error) {

	return nil, nil
}

func allPoliciesSQL(org *organization.Organization) ([]*Policy, error) {

	return nil, nil
}
