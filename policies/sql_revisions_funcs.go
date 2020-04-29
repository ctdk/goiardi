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
	"github.com/lib/pq"
	"golang.org/x/xerrors"
)

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

	pRevs := make([]*PolicyRevision, 0)
	for rows.Next() {
		pr := new(PolicyRevision)
		pr.pol = p
		err = pr.fillPolicyRevisionFromSQL(rows)
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

	return pRevs, nil
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

func (pr *PolicyRevision) saveRevisionSQL() error {
	prl, err := datastore.EncodeBlob(&pr.RunList)
	if err != nil {
		return err
	}

	pcl, err := datastore.EncodeBlob(&pr.CookbookLocks)
	if err != nil {
		return err
	}

	psd, err := datastore.EncodeBlob(&pr.SolutionDependencies)
	if err != nil {
		return err
	}

	pda, err := datastore.EncodeBlob(&pr.Default)
	if err != nil {
		return err
	}

	poa, err := datastore.EncodeBlob(&pr.Override)
	if err != nil {
		return err
	}

	tx, err := datastore.Dbh.Begin()
	if err != nil {
		return err
	}

	res, err := tx.Exec("INSERT INTO goiardi.policy_revisions (policy_id, revision_id, run_list, cookbook_locks, default_attr, override_attr, solution_dependencies, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())", pr.RevisionId, prl, pcl, pda, poa, psd)
	if err != nil {
		tx.Rollback()
		return err
	}

	pr.id, err = res.LastInsertId()
	if err != nil {
		tx.Rollback()
		return err
	}

	tx.Commit()

	return nil
}

func (pr *PolicyRevision) deleteRevisionSQL() error {
	tx, err := datastore.Dbh.Begin()
	if err != nil {
		return err
	}

	sqlStmt := "DELETE FROM goiardi.policy_revisions WHERE organization_id = $1 AND policy_id = $2 AND revision_id = $3"

	_, err = tx.Exec(sqlStmt, r.org.GetId(), pr.pol.id, pr.RevisionId)
	if err != nil {
		werr := xerrors.Errorf("deleting policy revision %s/%s had an error: %w", pr.PolicyName(), pr.RevisionId, err)
		terr := tx.Rollback()
		if terr != nil {
			werr = xerrors.Errorf("%s and then rolling back the transaction gave another error: %w", terr)
		}
		return werr
	}
	tx.Commit()
	return nil
}
