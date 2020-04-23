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
	"github.com/lib/pq"
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

	return p, nil
}

func (p *Policy) getRevisionSQL(revisionId string) (*PolicyRevision, error) {
	pr := new(PolicyRevision)
	pr.pol = p

	sqlStatement := "SELECT id, revision_id, run_list, cookbook_locks, default_attr, override_attr, solution_dependencies, created_at FROM goiardi.policy_revisions WHERE policy_id = $1 AND revision_id = $2"
	stmt, err := datastore.Dbh.Prepare(sqlStatement)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	row := stmt.QueryRow(p.id, revisionId)
	err = pr.fillPolicyRevisionFromSQL(row)
	if err != nil {
		return nil, err
	}

	return pr, nil
}

func (p *Policy) getAllRevisionsSQL() ([]*PolicyRevision, error) {
	sqlStatement := "SELECT id, revision_id, run_list, cookbook_locks, default_attr, override_attr, solution_dependencies, created_at FROM goiardi.policy_revisions WHERE policy_id = $1"
	stmt, err := datastore.Dbh.Prepare(sqlStatement)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.Query(p.id)
	if err != nil {
		return nil, err
	}

	pRevs := make([]*PolicyRevision)
	for rows.Next() {
		pr := new(PolicyRevision)
		pr.pol = p
		err = pr.fillPolicyRevisionFromSQL(row)
		if err != nil {
			rows.Close()
			return nil, err
		}
		pRevs = append(pRevs, pr)
	}

	rows.Close()
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return pr, nil
}

func (p *Policy) fillPolicyFromSQL(row datastore.ResRow) error {
	err := row.Scan(&p.id, &p.Name)
	if err != nil {
		return err
	}
	return nil
}

func (pr *PolicyRevision) fillPolicyRevisionFromSQL(row datastore.ResRow) error {
	var (
		rl []byte
		cl []byte
		sd []byte
		da []byte
		oa []byte
		ct pq.NullTime
	)

	err := row.Scan(&pr.id, &pr.RevisionId, &rl, &cl, &da, &oa, &sd, &ct)
	if err != nil {
		return err
	}
	if ct.Valid {
		pr.creationTime = ct.Time
	}

	if err = datastore.DecodeBlob(rl, &pr.RunList); err != nil {
		return err
	}
	if err = datastore.DecodeBlob(cl, &pr.CookbookLocks); err != nil {
		return err
	}
	if err = datastore.DecodeBlob(sd, &pr.SolutionDependencies); err != nil {
		return err
	}
	if err = datastore.DecodeBlob(da, &pr.Default); err != nil {
		return err
	}
	if err = datastore.DecodeBlob(oa, &pr.Override); err != nil {
		return err
	}

	return nil
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
