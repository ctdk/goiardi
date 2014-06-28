/*
 * Copyright (c) 2013-2014, Jeremy Bingham (<jbingham@gmail.com>)
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

/* Postgres specific functions for environments */

import (
	"github.com/ctdk/goiardi/data_store"
	"github.com/ctdk/goiardi/util"
)

func (e *ChefEnvironment) saveEnvironmentPostgreSQL() util.Gerror {
	dab, daerr := data_store.EncodeBlob(&e.Default)
	if daerr != nil {
		return util.CastErr(daerr)
	}
	oab, oaerr := data_store.EncodeBlob(&e.Override)
	if oaerr != nil {
		return util.CastErr(oaerr)
	}
	cvb, cverr := data_store.EncodeBlob(&e.CookbookVersions)
	if cverr != nil {
		return util.CastErr(cverr)
	}

	tx, err := data_store.Dbh.Begin()
	if err != nil {
		gerr := util.CastErr(err)
		return gerr
	}

	_, err = tx.Exec("SELECT goiardi.merge_environments($1, $2, $3, $4, $5)", e.Name, e.Description, dab, oab, cvb)
	if err != nil {
		tx.Rollback()
		gerr := util.CastErr(err)
		return gerr
	}
	tx.Commit()
	return nil
}
