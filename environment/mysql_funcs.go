/*
 * Copyright (c) 2013-2016, Jeremy Bingham (<jeremy@goiardi.gl>)
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

package environment

/* MySQL specific functions for environments */

import (
	"github.com/ctdk/goiardi/datastore"
	"github.com/ctdk/goiardi/util"
)

func (e *ChefEnvironment) saveEnvironmentMySQL() util.Gerror {
	dab, daerr := datastore.EncodeBlob(&e.Default)
	if daerr != nil {
		return util.CastErr(daerr)
	}
	oab, oaerr := datastore.EncodeBlob(&e.Override)
	if oaerr != nil {
		return util.CastErr(oaerr)
	}
	cvb, cverr := datastore.EncodeBlob(&e.CookbookVersions)
	if cverr != nil {
		return util.CastErr(cverr)
	}

	tx, err := datastore.Dbh.Begin()
	if err != nil {
		return util.CastErr(err)
	}

	_, err = tx.Exec("INSERT INTO environments (name, description, default_attr, override_attr, cookbook_vers, created_at, updated_at) VALUES (?, ?, ?, ?, ?, NOW(), NOW()) ON DUPLICATE KEY UPDATE description = ?, default_attr = ?, override_attr = ?, cookbook_vers = ?, updated_at = NOW()", e.Name, e.Description, dab, oab, cvb, e.Description, dab, oab, cvb)
	if err != nil {
		tx.Rollback()
		gerr := util.CastErr(err)
		return gerr
	}

	tx.Commit()
	return nil
}
