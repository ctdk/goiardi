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

// MySQL specific functions for cookbooks

package cookbook

import (
	"github.com/ctdk/goiardi/data_store"
	"net/http"
	"github.com/ctdk/goiardi/util"
)

func (c *Cookbook) saveCookbookMySQL() error {
	tx, err := data_store.Dbh.Begin()
	if err != nil {
		return err
	}
	res, err := tx.Exec("INSERT INTO cookbooks (name, created_at, updated_at) VALUES (?, NOW(), NOW()) ON DUPLICATE KEY UPDATE id=LAST_INSERT_ID(id), name = ?, updated_at = NOW()", c.Name, c.Name)
	if err != nil {
		tx.Rollback()
		return err
	}
	c_id, err := res.LastInsertId()
	if err != nil {
		tx.Rollback()
		return err
	}
	c.id = int32(c_id)
	
	tx.Commit()
	return nil
}



func (cbv *CookbookVersion) updateCookbookVersionMySQL(defb, libb, attb, recb, prob, resb, temb, roob, filb, metb []byte, maj, min, patch int64) util.Gerror {
	tx, err := data_store.Dbh.Begin()
	if err != nil {
		gerr := util.Errorf(err.Error())
		gerr.SetStatus(http.StatusInternalServerError)
		return gerr
	}

	res, err := tx.Exec("INSERT INTO cookbook_versions (cookbook_id, major_ver, minor_ver, patch_ver, frozen, metadata, definitions, libraries, attributes, recipes, providers, resources, templates, root_files, files, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NOW(), NOW()) ON DUPLICATE KEY UPDATE id=LAST_INSERT_ID(id), frozen = ?, metadata = ?, definitions = ?, libraries = ?, attributes = ?, recipes = ?, providers = ?, resources = ?, templates = ?, root_files = ?, files = ?, updated_at = NOW()", cbv.cookbook_id, maj, min, patch, cbv.IsFrozen, metb, defb, libb, attb, recb, prob, resb, temb, roob, filb, cbv.IsFrozen, metb, defb, libb, attb, recb, prob, resb, temb, roob, filb)
	if err != nil {
		tx.Rollback()
		gerr := util.CastErr(err)
		gerr.SetStatus(http.StatusInternalServerError)
		return gerr
	}
	c_id, err := res.LastInsertId()
	if err != nil {
		tx.Rollback()
		gerr := util.Errorf(err.Error())
		gerr.SetStatus(http.StatusInternalServerError)
		return gerr
	}
	cbv.id = int32(c_id)

	tx.Commit()
	return nil
}
