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
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/datastore"
	"github.com/ctdk/goiardi/organization"
	"github.com/ctdk/goiardi/user"
	"github.com/ctdk/goiardi/util"
	"log"
)

func checkForAssociationSQL(dbhandle datastore.Dbhandle, user *user.User, org *organization.Organization) (bool, util.Gerror) {
	var z int
	var sqlStmt string
	if config.Config.UseMySQL {
		// come back if we decide to actually keep mysql still - it's
		// iffy
	} else if config.Config.UsePostgreSQL {
		sqlStmt = "SELECT count(*) AS c FROM goiardi.associations WHERE user_id = $1 AND organization_id = $2"
	}

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
	var sqlStmt string
	if config.Config.UseMySQL {
		// come back if we decide to actually keep mysql still - it's
		// iffy
	} else if config.Config.UsePostgreSQL {
		sqlStmt = "SELECT count(*) AS c FROM goiardi.association_requests WHERE user_id = $1 AND organization_id = $2 AND inviter_id = $3 AND inviter_type = $4"
	}

	stmt, err := dbhandle.Prepare(sqlStmt)
	if err != nil {
		return false, util.CastErr(err)
	}
	defer stmt.Close()
	err = stmt.QueryRow(user.GetId(), org.GetId(), inviter.GetId(), inviter.URLType()).Scan(&z)
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

func (a *Association) fillAssociationFromSQL(row datastore.ResRow) util.Gerror {
	// add mysql later if we do that
	return a.fillAssociationFromPostgreSQL(row)
}

func getAssociationSQL(user *user.User, org *organization.Organization) (*Association, util.Gerror) {
	a := new(Association)

	var sqlStmt string
	if config.Config.UseMySQL {
		// mebbe?
	} else if config.Config.UsePostgreSQL {
		sqlStmt = "SELECT u.name AS user_name, o.name AS org_name FROM goiardi.assocations assoc LEFT JOIN goiardi.users u ON assoc.user_id = u.id LEFT JOIN goiardi.organizations o ON assoc.organization_id = o.id WHERE u.id = $1 AND o.id = $2"
	}

	stmt, err := datastore.Dbh.Prepare(sqlStmt)
	if err != nil {
		return nil, util.CastErr(err)
	}
	defer stmt.Close()

	row := stmt.QueryRow(user.GetId(), org.GetId())
	if err = a.fillAssociationFromSQL(row); err != nil {
		return nil, util.CastErr(err)
	}

	return a, nil
}

func (a *AssociationReq) fillAssociationReqFromSQL(row datastore.ResRow) util.Gerror {
	// add mysql later if we do that
	return a.fillAssociationReqFromPostgreSQL(row)
}

func getAssociationReqSQL(user *user.User, org *organization.Organization, inviter actor.Actor, status string) (*AssociationReq, util.Gerror) {
	a := new(AssociationReq)
	
	var sqlStmt string
	if config.Config.UseMySQL {
		// mebbe?
	} else if config.Config.UsePostgreSQL {
		sqlStmt = fmt.Sprintf("SELECT u.name AS user_name, o.name AS org_name, i.name AS inviter_name FROM goiardi.assocations assoc LEFT JOIN goiardi.users u ON assoc.user_id = u.id LEFT JOIN goiardi.organizations o ON assoc.organization_id = o.id LEFT JOIN goiardi.%s i ON assoc.inviter_id = i.id WHERE u.id = $1 AND org.id = $2 AND inviter_name = $3 AND assoc.inviter_type = $4 AND assoc.status = $5", inviter.URLType())
	}

	stmt, err := datastore.Dbh.Prepare(sqlStmt)
	if err != nil {
		return nil, util.CastErr(err)
	}
	defer stmt.Close()

	row := stmt.QueryRow(user.GetId(), org.GetId(), inviter.GetName(), inviter.URLType(), status)
	if err = a.fillAssociationReqFromSQL(row); err != nil {
		return nil, util.CastErr(err)
	}

	return a, nil
}

// may be extraneous
func (a *Association) saveSQL() util.Gerror {

}

func (a *Association) deleteSQL() util.Gerror {
	
}

func userAssociationsSQL(org *organization.Organization) ([]*user.User, util.Gerror) {

}

func orgAssociationsSQL(user *user.User) ([]*organization.Organization, util.Gerror) {

}

func deleteAllOrgAssociationsSQL(org *organization.Organization) util.Gerror {

}

func deleteAllUserAssociationsSQL(user *user.User) util.Gerror {

}

func (a *AssociationReq) acceptSQL() util.Gerror {
	if config.Config.UseMySQL {
		return nil
	} else {
		return a.acceptPostgresSQL()
	}
}

func (a *AssociationReq) rejectSQL() util.Gerror {
	a.Status = 'rejected'
	return a.savePostgreSQL()
}

func (a *AssociationReq) deleteSQL() util.Gerror {
	var sqlStmt string
	if config.Config.UseMySQL {

	} else {
		sqlStmt = "DELETE FROM goiardi.association_requests WHERE user_id = $1 AND organization_id = $2 AND inviter_id = $3 AND inviter_type = $4"
	}

	tx, err := datastore.Dbh.Begin()
	if err != nil {
		return util.CastErr(err)
	}
	_, err = tx.Exec(sqlStmt, a.User.GetId(), a.Org.GetId(), a.Inviter.GetId(), a.Inviter.URLType(), a.Status)

	if err != nil {
		tx.Rollback()
		return util.CastErr(err)
	}
	tx.Commit()
	return nil
}

func orgsAssociationReqCountSQL(user *user.User) (int, util.Gerror) {
	var c int
	var sqlStmt string
	// deal with mysql if/when later
	if config.Config.UseMySQL {

	} else {
		sqlStmt = "SELECT COUNT(*) c FROM goiardi.association_requests WHERE user_id = $1"
	}

	stmt, err := dbhandle.Prepare(sqlStmt)
	if err != nil {
		return false, util.CastErr(err)
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
	var sqlStmt string
	// deal with mysql if/when later
	if config.Config.UseMySQL {

	} else {
		sqlStmt = "SELECT COUNT(*) c FROM goiardi.association_requests WHERE organization_id = $1"
	}

	stmt, err := dbhandle.Prepare(sqlStmt)
	if err != nil {
		return false, util.CastErr(err)
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

}

func getUserAssociationReqsSQL(org *organization.Organization) ([]*AssociationReq, util.Gerror) {

}

func deleteUserAssociationReqsSQL(user *user.User) util.Gerror {

}

func deleteOrgAssociationReqsSQL(org *organization.Organization) util.Gerror {

}

func (a *AssociationReq) saveSQL() util.Gerror {
	if config.Config.UseMySQL {
		return fmt.Errorf("MySQL's not implemented for this yet")
	} else if config.Config.UsePostgreSQL {
		return a.savePostgreSQL()
	}
}
