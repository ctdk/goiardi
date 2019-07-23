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
	"database/sql"
	"fmt"
	"github.com/ctdk/goiardi/actor"
	"github.com/ctdk/goiardi/client"
	"github.com/ctdk/goiardi/datastore"
	"github.com/ctdk/goiardi/organization"
	"github.com/ctdk/goiardi/orgloader"
	"github.com/ctdk/goiardi/user"
	"github.com/ctdk/goiardi/util"
	"net/http"
)

func checkForAssociationSQL(dbhandle datastore.Dbhandle, user *user.User, org *organization.Organization) (bool, util.Gerror) {
	var z int
	sqlStmt := "SELECT count(*) AS c FROM goiardi.associations WHERE user_id = $1 AND organization_id = $2"

	stmt, err := dbhandle.Prepare(sqlStmt)
	if err != nil {
		return false, util.CastErr(err)
	}
	defer stmt.Close()
	err = stmt.QueryRow(user.GetId(), org.GetId()).Scan(&z)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, util.CastErr(err)
	}
	if z > 0 {
		return true, nil
	}
	return false, nil
}

func checkForAssociationReqSQL(dbhandle datastore.Dbhandle, user *user.User, org *organization.Organization, inviter actor.Actor) (bool, util.Gerror) {
	var z int
	sqlStmt := "SELECT count(*) AS c FROM goiardi.association_requests WHERE user_id = $1 AND organization_id = $2 AND inviter_id = $3 AND inviter_type = $4 AND status = 'pending'"

	stmt, err := dbhandle.Prepare(sqlStmt)
	if err != nil {
		return false, util.CastErr(err)
	}
	defer stmt.Close()
	err = stmt.QueryRow(user.GetId(), org.GetId(), inviter.GetId(), inviterType(inviter)).Scan(&z)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, util.CastErr(err)
	}
	if z > 0 {
		return true, nil
	}
	return false, nil
}

func (a *Association) fillAssociationFromSQL(row datastore.ResRow) error {
	// add mysql later if we do that
	return a.fillAssociationFromPostgreSQL(row)
}

func getAssociationSQL(user *user.User, org *organization.Organization) (*Association, util.Gerror) {
	a := new(Association)
	sqlStmt := "SELECT u.name AS user_name, o.name AS org_name FROM goiardi.associations assoc LEFT JOIN goiardi.users u ON assoc.user_id = u.id LEFT JOIN goiardi.organizations o ON assoc.organization_id = o.id WHERE u.id = $1 AND o.id = $2"

	stmt, err := datastore.Dbh.Prepare(sqlStmt)
	if err != nil {
		return nil, util.CastErr(err)
	}
	defer stmt.Close()

	row := stmt.QueryRow(user.GetId(), org.GetId())
	if err = a.fillAssociationFromSQL(row); err != nil {
		if err != sql.ErrNoRows {
			return nil, util.CastErr(err)
		}
		gerr := util.Errorf("'%s' not associated with organization '%s'", user.Name, org.Name)
		gerr.SetStatus(http.StatusForbidden)
		return nil, gerr
	}

	return a, nil
}

func (a *AssociationReq) fillAssociationReqFromSQL(row datastore.ResRow) util.Gerror {
	// add mysql later if we do that
	return a.fillAssociationReqFromPostgreSQL(row)
}

func getAssociationReqSQL(uReq *user.User, org *organization.Organization) (*AssociationReq, util.Gerror) {
	sqlStmt := "SELECT inviter_id, inviter_type FROM goiardi.association_requests WHERE user_id = $1 AND organization_id = $2 AND status = 'pending' ORDER BY id DESC LIMIT 1"

	stmt, err := datastore.Dbh.Prepare(sqlStmt)
	if err != nil {
		return nil, util.CastErr(err)
	}
	defer stmt.Close()

	var inviterId int64
	var inviterType string

	row := stmt.QueryRow(uReq.GetId(), org.GetId())
	if err = row.Scan(&inviterId, &inviterType); err != nil {
		return nil, util.CastErr(err)
	}

	var inviter actor.Actor

	switch inviterType {
	case "users":
		u, uerr := user.UsersByIdSQL([]int64{inviterId})
		if uerr != nil {
			return nil, util.CastErr(uerr)
		}
		if u == nil || len(u) == 0 {
			return nil, util.Errorf("Inviter user id %d not found", inviterId)
		}
		inviter = u[0]
	case "clients":
		c, cerr := client.ClientsByIdSQL([]int64{inviterId}, org)
		if cerr != nil {
			return nil, util.CastErr(cerr)
		}
		if c == nil || len(c) == 0 {
			return nil, util.Errorf("Inviter client id %d not found", inviterId)
		}
		inviter = c[0]
	default:
		return nil, util.Errorf("Unknown inviter type '%s'.", inviterType)
	}

	return getExactAssociationReqSQL(uReq, org, inviter, "pending")
}

func getExactAssociationReqSQL(user *user.User, org *organization.Organization, inviter actor.Actor, status string) (*AssociationReq, util.Gerror) {
	a := new(AssociationReq)

	sqlStmt := fmt.Sprintf("SELECT u.name AS user_name, o.name AS org_name, i.name AS inviter_name, status FROM goiardi.association_requests assoc LEFT JOIN goiardi.users u ON assoc.user_id = u.id LEFT JOIN goiardi.organizations o ON assoc.organization_id = o.id LEFT JOIN goiardi.%s i ON assoc.inviter_id = i.id WHERE u.id = $1 AND o.id = $2 AND i.name = $3 AND assoc.inviter_type = $4 AND assoc.status = $5", inviterType(inviter))

	stmt, err := datastore.Dbh.Prepare(sqlStmt)
	if err != nil {
		return nil, util.CastErr(err)
	}
	defer stmt.Close()

	row := stmt.QueryRow(user.GetId(), org.GetId(), inviter.GetName(), inviterType(inviter), status)
	if err = a.fillAssociationReqFromSQL(row); err != nil {
		return nil, util.CastErr(err)
	}

	return a, nil
}

// At this time there doesn't seem to be a need for a saveSQL function for
// associations - aside from creating it in the first place and possibly
// deleting them, there's not much to edit.

func (a *Association) deleteSQL() util.Gerror {
	sqlStmt := "DELETE FROM goiardi.associations WHERE user_id = $1 AND organization_id = $2"

	tx, err := datastore.Dbh.Begin()
	if err != nil {
		return util.CastErr(err)
	}
	_, err = tx.Exec(sqlStmt, a.User.GetId(), a.Org.GetId())

	if err != nil {
		tx.Rollback()
		return util.CastErr(err)
	}
	tx.Commit()

	return nil
}

func userAssociationsSQL(org *organization.Organization) ([]*user.User, util.Gerror) {
	sqlStmt := "SELECT user_id FROM goiardi.associations WHERE organization_id = $1"

	stmt, err := datastore.Dbh.Prepare(sqlStmt)
	if err != nil {
		return nil, util.CastErr(err)
	}
	defer stmt.Close()

	rows, err := stmt.Query(org.GetId())
	if err != nil {
		if err == sql.ErrNoRows {
			ua := make([]*user.User, 0) // eh?
			return ua, nil
		}
		return nil, util.CastErr(err)
	}
	userIds := make([]int64, 0)
	for rows.Next() {
		var i int64
		ierr := rows.Scan(&i)
		if ierr != nil {
			return nil, util.CastErr(ierr)
		}
		userIds = append(userIds, i)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return nil, util.CastErr(err)
	}

	userAssoc, err := user.UsersByIdSQL(userIds)

	if err != nil {
		return nil, util.CastErr(err)
	}

	return userAssoc, nil
}

func orgAssociationsSQL(u *user.User) ([]*organization.Organization, util.Gerror) {
	sqlStmt := "SELECT organization_id FROM goiardi.associations WHERE user_id = $1"

	stmt, err := datastore.Dbh.Prepare(sqlStmt)
	if err != nil {
		return nil, util.CastErr(err)
	}
	defer stmt.Close()

	rows, err := stmt.Query(u.GetId())
	if err != nil {
		if err == sql.ErrNoRows {
			oa := make([]*organization.Organization, 0) // eh?
			return oa, nil
		}
		return nil, util.CastErr(err)
	}
	orgIds := make([]int64, 0)
	for rows.Next() {
		var i int64
		ierr := rows.Scan(&i)
		if ierr != nil {
			return nil, util.CastErr(ierr)
		}
		orgIds = append(orgIds, i)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return nil, util.CastErr(err)
	}

	if len(orgIds) == 0 {
		return nil, nil
	}

	orgAssoc, err := orgloader.OrgsByIdSQL(orgIds)

	if err != nil {
		return nil, util.CastErr(err)
	}

	return orgAssoc, nil
}

func deleteAllOrgAssociationsSQL(org *organization.Organization) util.Gerror {
	sqlStmt := "DELETE FROM goiardi.associations WHERE organization_id = $1"

	tx, err := datastore.Dbh.Begin()
	if err != nil {
		return util.CastErr(err)
	}
	_, err = tx.Exec(sqlStmt, org.GetId())

	if err != nil {
		tx.Rollback()
		return util.CastErr(err)
	}
	tx.Commit()

	return nil
}

func deleteAllUserAssociationsSQL(u *user.User) util.Gerror {
	sqlStmt := "DELETE FROM goiardi.associations WHERE user_id = $1"

	tx, err := datastore.Dbh.Begin()
	if err != nil {
		return util.CastErr(err)
	}
	_, err = tx.Exec(sqlStmt, u.GetId())

	if err != nil {
		tx.Rollback()
		return util.CastErr(err)
	}
	tx.Commit()

	return nil
}

func (a *AssociationReq) acceptSQL() util.Gerror {
	return a.acceptPostgreSQL()
}

func (a *AssociationReq) rejectSQL() util.Gerror {
	a.Status = "rejected"
	return a.savePostgreSQL()
}

func (a *AssociationReq) deleteSQL() util.Gerror {
	sqlStmt := "DELETE FROM goiardi.association_requests WHERE user_id = $1 AND organization_id = $2 AND inviter_id = $3 AND inviter_type = $4"

	tx, err := datastore.Dbh.Begin()
	if err != nil {
		return util.CastErr(err)
	}
	_, err = tx.Exec(sqlStmt, a.User.GetId(), a.Org.GetId(), a.Inviter.GetId(), inviterType(a.Inviter), a.Status)

	if err != nil {
		tx.Rollback()
		return util.CastErr(err)
	}
	tx.Commit()
	return nil
}

func orgsAssociationReqCountSQL(user *user.User) (int, util.Gerror) {
	var c int
	sqlStmt := "SELECT COUNT(*) c FROM goiardi.association_requests WHERE user_id = $1"

	stmt, err := datastore.Dbh.Prepare(sqlStmt)
	if err != nil {
		return 0, util.CastErr(err)
	}
	defer stmt.Close()
	err = stmt.QueryRow(user.GetId()).Scan(&c)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, util.CastErr(err)
	}
	return c, nil
}

func userAssociationReqCountSQL(org *organization.Organization) (int, util.Gerror) {
	var c int
	sqlStmt := "SELECT COUNT(*) c FROM goiardi.association_requests WHERE organization_id = $1"

	stmt, err := datastore.Dbh.Prepare(sqlStmt)
	if err != nil {
		return 0, util.CastErr(err)
	}
	defer stmt.Close()
	err = stmt.QueryRow(org.GetId()).Scan(&c)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, util.CastErr(err)
	}
	return c, nil
}

func getOrgAssociationReqsSQL(user *user.User) ([]*AssociationReq, util.Gerror) {
	sqlStmt := "SELECT u.name AS user_name, o.name AS org_name, i.name AS inviter_name FROM goiardi.association_requests assoc LEFT JOIN goiardi.users u ON assoc.user_id = u.id LEFT JOIN goiardi.organizations o ON assoc.organization_id = o.id LEFT JOIN goiardi.%s i ON assoc.inviter_id = i.id WHERE user_id = $1"

	stmt, err := datastore.Dbh.Prepare(sqlStmt)
	if err != nil {
		return nil, util.CastErr(err)
	}
	defer stmt.Close()

	oar := make([]*AssociationReq, 0)

	rows, err := stmt.Query(user.GetId())
	if err != nil {
		if err == sql.ErrNoRows {
			return oar, nil
		}
		return nil, util.CastErr(err)
	}
	for rows.Next() {
		ar := new(AssociationReq)
		if err = ar.fillAssociationReqFromSQL(rows); err != nil {
			return nil, util.CastErr(err)
		}
		oar = append(oar, ar)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return nil, util.CastErr(err)
	}

	return oar, nil
}

func getUserAssociationReqsSQL(org *organization.Organization) ([]*AssociationReq, util.Gerror) {
	sqlStmt := "SELECT u.name AS user_name, o.name AS org_name, i.name AS inviter_name FROM goiardi.association_requests assoc LEFT JOIN goiardi.users u ON assoc.user_id = u.id LEFT JOIN goiardi.organizations o ON assoc.organization_id = o.id LEFT JOIN goiardi.%s i ON assoc.inviter_id = i.id WHERE organization_id = $1"

	stmt, err := datastore.Dbh.Prepare(sqlStmt)
	if err != nil {
		return nil, util.CastErr(err)
	}
	defer stmt.Close()

	uar := make([]*AssociationReq, 0)

	rows, err := stmt.Query(org.GetId())
	if err != nil {
		if err == sql.ErrNoRows {
			return uar, nil
		}
		return nil, util.CastErr(err)
	}
	for rows.Next() {
		ar := new(AssociationReq)
		if err = ar.fillAssociationReqFromSQL(rows); err != nil {
			return nil, util.CastErr(err)
		}
		uar = append(uar, ar)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return nil, util.CastErr(err)
	}

	return uar, nil
}

func deleteUserAssociationReqsSQL(user *user.User) util.Gerror {
	sqlStmt := "DELETE FROM goiardi.association_requests WHERE user_id = $1"

	tx, err := datastore.Dbh.Begin()
	if err != nil {
		return util.CastErr(err)
	}
	_, err = tx.Exec(sqlStmt, user.GetId())

	if err != nil {
		tx.Rollback()
		return util.CastErr(err)
	}
	tx.Commit()
	return nil
}

func deleteOrgAssociationReqsSQL(org *organization.Organization) util.Gerror {
	sqlStmt := "DELETE FROM goiardi.association_requests WHERE organization_id = $1"

	tx, err := datastore.Dbh.Begin()
	if err != nil {
		return util.CastErr(err)
	}
	_, err = tx.Exec(sqlStmt, org.GetId())

	if err != nil {
		tx.Rollback()
		return util.CastErr(err)
	}
	tx.Commit()
	return nil
}

func (a *AssociationReq) saveSQL() util.Gerror {
	return a.savePostgreSQL()
}

func inviterType(inviter actor.Actor) string {
	var iT string
	if inviter.IsUser() {
		iT = "users"
	} else {
		iT = "clients"
	}
	return iT
}
