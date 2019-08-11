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
	"github.com/ctdk/goiardi/actor"
	"github.com/ctdk/goiardi/client"
	"github.com/ctdk/goiardi/datastore"
	"github.com/ctdk/goiardi/orgloader"
	"github.com/ctdk/goiardi/user"
	"github.com/ctdk/goiardi/util"
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

func (a *AssociationReq) fillAssociationReqFromPostgreSQL(row datastore.ResRow) util.Gerror {
	var userName, orgName, inviterType string
	var inviterId int64

	if err := row.Scan(&userName, &orgName, &inviterId, &inviterType, &a.Status); err != nil {
		return util.CastErr(err)
	}

	// fill in the users and org now
	u, err := user.Get(userName)
	if err != nil {
		return util.CastErr(err)
	}
	a.User = u

	o, err := orgloader.Get(orgName)
	if err != nil {
		return util.CastErr(err)
	}
	a.Org = o

	var invitinator actor.Actor

	switch inviterType {
	case "users":
		u, err := user.UsersByIdSQL([]int64{inviterId})
		if err != nil {
			return util.CastErr(err)
		}
		if len(u) > 0 {
			invitinator = u[0]
		}
	case "clients":
		c, err := client.ClientsByIdSQL([]int64{inviterId}, o)
		if err != nil {
			return util.CastErr(err)
		}
		if len(c) > 0 {
			invitinator = c[0]
		}
	}

	a.Inviter = invitinator

	return nil
}

func (a *AssociationReq) savePostgreSQL() util.Gerror {
	tx, err := datastore.Dbh.Begin()
	if err != nil {
		return util.CastErr(err)
	}

	_, err = tx.Exec("SELECT goiardi.merge_association_requests($1, $2, $3, $4, $5)", a.User.GetId(), a.Org.GetId(), a.Inviter.GetId(), inviterType(a.Inviter), a.Status)

	if err != nil {
		tx.Rollback()
		return util.CastErr(err)
	}
	tx.Commit()
	return nil
}

func (a *AssociationReq) acceptPostgreSQL() util.Gerror {
	a.Status = "accepted"

	tx, err := datastore.Dbh.Begin()
	if err != nil {
		return util.CastErr(err)
	}
	_, err = tx.Exec("SELECT goiardi.merge_association_requests($1, $2, $3, $4, $5)", a.User.GetId(), a.Org.GetId(), a.Inviter.GetId(), inviterType(a.Inviter), a.Status)

	if err != nil {
		tx.Rollback()
		return util.CastErr(err)
	}

	_, err = tx.Exec("INSERT INTO goiardi.associations(user_id, organization_id, association_request_id, created_at, updated_at) VALUES($1, $2, $3, NOW(), NOW())", a.User.GetId(), a.Org.GetId(), a.id)

	if err != nil {
		tx.Rollback()
		return util.CastErr(err)
	}

	tx.Commit()
	return nil
}
