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

// Postgres specific functions for cookbooks

package cookbook

import (
	"github.com/ctdk/goiardi/data_store"
	"github.com/ctdk/goiardi/util"
	"net/http"
)

func (c *Cookbook)saveCookbookPostgreSQL() error {
	tx, err := data_store.Dbh.Begin()
	if err != nil {
		return err
	}
	
	err = tx.QueryRow("SELECT goiardi.merge_cookbooks($1)", c.Name).Scan(&c.id)
	if err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()
	return nil
}

func (cbv *CookbookVersion) updateCookbookVersionPostgreSQL(defb, libb, attb, recb, prob, resb, temb, roob, filb, metb []byte, maj, min, patch int64) util.Gerror {
	tx, err := data_store.Dbh.Begin()
	if err != nil {
		gerr := util.Errorf(err.Error())
		gerr.SetStatus(http.StatusInternalServerError)
		return gerr
	}
	err = tx.QueryRow("SELECT goiardi.merge_cookbook_versions($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)", cbv.cookbook_id, cbv.IsFrozen, defb, libb, attb, recb, prob, resb, temb, roob, filb, metb, maj, min, patch).Scan(&cbv.id)
	if err != nil {
		tx.Rollback()
		gerr := util.CastErr(err)
		gerr.SetStatus(http.StatusInternalServerError)
		return gerr
	}
	tx.Commit()
	return nil
}
