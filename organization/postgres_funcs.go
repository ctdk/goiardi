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

package organization

import (
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/datastore"
	"github.com/ctdk/goiardi/util"
)

/*
 * Postgres specific functions for organizations.
 */

func (o *Organization) savePostgreSQL() util.Gerror {
	var pk string
	if config.Config.UseShovey && !config.Config.UseExtSecrets {
		var err error
		if pk, err = o.ShoveyPrivKey(); err != nil {
			return util.CastErr(err)
		}
	}
	
	tx, err := datastore.Dbh.Begin()
	if err != nil {
		return util.CastErr(err)
	}

	// Looks like we've got some postgres functions to update
	err = tx.QueryRow("SELECT goiardi.merge_orgs($1, $2, $3, $4)", o.Name, o.FullName, o.GUID, o.uuID).Scan(&o.id)

	if err != nil {
		tx.Rollback()
		return util.CastErr(err)
	}
	tx.Commit()
	return nil
}

func (o *Organization) renamePostgreSQL(newName string) util.Gerror {
	tx, err := datastore.Dbh.Begin()
	if err != nil {
		return util.CastErr(err)
	}

	_, err = tx.Exec("UPDATE goiardi.organizations SET name = $1 WHERE id = $2", newName, o.id)

	// do we need to set o.Name here, or is that taken care of further up?

	if err != nil {
		tx.Rollback()
		return util.CastErr(err)
	}
	tx.Commit()
	return nil
}

func (o *Organization) createSearchSchema() util.Gerror {
	tx, err := datastore.Dbh.Begin()
	if err != nil {
		return util.CastErr(err)
	}

	_, err = tx.Exec("SELECT goiardi.clone_schema($1, $2)", util.BaseSearchSchema, o.SearchSchemaName)

	if err != nil {
		tx.Rollback()
		return util.CastErr(err)
	}
	tx.Commit()
	return nil
}
