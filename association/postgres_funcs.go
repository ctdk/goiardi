/*
 * Copyright (c) 2013-2019, Jeremy Bingham (<jbingham@gmail.com>)
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

package association

import (
	"github.com/ctdk/goiardi/datastore"
	"github.com/ctdk/goiardi/orgloader"
	"github.com/ctdk/goiardi/user"
	"github.com/lib/pq"
)

func (a *Association) fillAssociationFromPostgreSQL(row datastore.ResRow) error {
	var userName, orgName string
	
	err := row.Scan(&userName, &orgName)
	if err != nil {
		return err
	}

	// fill in the user and org now
	u, err := user.Get(userName)
	if err != nil {
		return err
	}
	a.User = u

	o, err := orgloader.Get(orgName)
	if err != nil {
		return err
	}
	a.Org = o

	return nil
}

func (a *AssociationReq) fillAssociationReqFromPostgreSQL(row datastore.ResRow) error {
	var userName, orgName, inviterName string
	
	err := row.Scan(&userName, &orgName, &inviterName, &a.Status)
	if err != nil {
		return err
	}

	// fill in the users and org now
	u, err := user.Get(userName)
	if err != nil {
		return err
	}
	a.User = u

	o, err := orgloader.Get(orgName)
	if err != nil {
		return err
	}
	a.Org = o

	i, err := actor.GetActor(inviterName)
	if err != nil {
		return err
	}
	a.Inviter = i

	return nil
}

func (a *AssociationReq) savePostgreSQL() error {
	tx, err := datastore.Dbh.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec("SELECT goiardi.merge_association_requests($1, $2, $3, $4, $5)", a.User.GetId(), a.Org.GetId(), a.Inviter.GetId(), a.Inviter.URLType(), a.Status)

	if err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()
	return nil
}

func (a *AssociationReq) acceptPostgresSQL() error {
	a.Status = 'accepted'

	tx, err := datastore.Dbh.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec("SELECT goiardi.merge_association_requests($1, $2, $3, $4, $5)", a.User.GetId(), a.Org.GetId(), a.Inviter.GetId(), a.Inviter.URLType(), a.Status)

	if err != nil {
		tx.Rollback()
		return err
	}

	_, err = tx.Exec("INSERT INTO goiardi.associations(user_id, organization_id, association_request_id, created_at, updated_at) VALUES($1, $2, $3, NOW(), NOW())", a.User.GetId(), a.Org.GetId(), a.id)

	if err != nil {
		tx.Rollback()
		return err
	}

	tx.Commit()
	return nil
}
