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

func (p *Policy) checkForRevisionSQL(dbhandle datastore.Dbhandle, revisionId string) (bool, error) {
	var found bool

	sqlStatement := "SELECT COUNT(pr.id) FROM goiardi.policy_revisions pr LEFT JOIN goiardi.policies p ON pr.policy_id = p.id WHERE pr.policy_id = $1 AND pr.revision_id = $2 AND p.organization_id = $3"

	stmt, err := datastore.Dbh.Prepare(sqlStatement)
	if err != nil {
		return false, err
	}
	defer stmt.Close()

	var cn int
	err = stmt.QueryRow(p.id, revisionId, p.org.GetId()).Scan(&cn)
	if err != nil && !xerrors.Is(err, sql.ErrNoRows) {
		return false, err
	}
	if cn != 0 {
		found = true
	}

	return found, nil
}

func getPolicySQL(org *organization.Organization, name string) (*Policy, error) {

	return nil, nil
}

func (p *Policy) savePolicySQL() error {

	return nil
}

func (p *Policy) deletePolicySQL() error {

	return nil
}

func getListSQL(org *organization.Organization) ([]string, error) {

	return nil, nil
}

func allPoliciesSQL(org *organization.Organization) ([]*Policy, error) {

	return nil, nil
}
