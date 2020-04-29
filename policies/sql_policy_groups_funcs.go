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

func checkForPolicyGroupSQL(dbhandle datastore.Dbhandle, org *organization.Organization, name string) (bool, error) {
	_, err := datastore.CheckForOne(dbhandle, "policy_groups", org.GetId(), name)
	if err == nil {
		return true, nil
	}

	if !xerrors.Is(err, sql.ErrNoRows) {
		return false, err
	}
	return false, nil
}

func getPolicyGroupSQL(org *organization.Organization, name string) (*PolicyGroup, error) {
	pg := new(PolicyGroup)
	pg.org = org
	
	// hoo-boy, this is interesting
	sqlStatement := `SELECT id, name,
		(SELECT array_to_json(COALESCE(ARRAY_AGG(row_to_json(j)), ARRAY[]::json[]))
			FROM (
				SELECT policy_id, policy_rev_id, p.name, pr.revision_id
					FROM goiardi.policy_groups_to_policies pgp
					LEFT JOIN goiardi.policy_revisions pr
						ON pgp.policy_rev_id = pr.id
					LEFT JOIN goiardi.policies p
						ON pgp.policy_id = p.id
					WHERE pg_id = policy_groups.id
			) AS j
		) AS rev_json
		FROM goiardi.policy_groups pg
		WHERE organization_id = $1 AND name = $2`
	stmt, err := datastore.Dbh.Prepare(sqlStatement)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	row := stmt.QueryRow(org.GetId(), name)
	err = pg.fillPolicyGroupFromSQL(row)
	if err != nil {
		return nil, err
	}

	return pg, nil
}

func (pg *PolicyGroup) fillPolicyGroupFromSQL(row datastore.ResRow) error {
	var pgp []byte
	var revJSON []*pgRevisionInfo

	err := row.Scan(&pg.id, &pg.Name, &pgp)
	if err != nil {
		return err
	}

	// holding my breath...
	if err = datastore.DecodeBlob(pgp, &revJSON); err != nil {
		return err
	}
	// if that worked...
	rm := make(map[string]*pgRevisionInfo, len(revJSON))
	for _, v := range revJSON {
		rm[v.PolicyName] = v
	}
	pg.policyInfo = rm

	return nil
}
