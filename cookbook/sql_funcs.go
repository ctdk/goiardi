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

// Common SQL functions for cookbooks

package cookbook

import (
	"github.com/ctdk/goiardi/util"
	"github.com/ctdk/goiardi/data_store"
	"database/sql"
	"github.com/ctdk/goiardi/config"
	"fmt"
	"log"
	"net/http"
	"sort"
)

func (c *Cookbook)numVersionsSQL() *int {
	var cbv_count int
	var sqlStatement string
	if config.Config.UseMySQL {
		sqlStatement = "SELECT count(*) AS c FROM cookbook_versions cbv WHERE cbv.cookbook_id = ?"
	} else if config.Config.UsePostgreSQL {
		sqlStatement = "SELECT count(*) AS c FROM goiardi.cookbook_versions cbv WHERE cbv.cookbook_id = $1"
	}
	stmt, err := data_store.Dbh.Prepare(sqlStatement)

	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()
	err = stmt.QueryRow(c.id).Scan(&cbv_count)
	if err != nil {
		if err == sql.ErrNoRows {
			cbv_count = 0
		} else {
			log.Fatal(err)
		}
	}
	return &cbv_count
}

func checkForCookbookSQL(dbhandle data_store.Dbhandle, name string) (bool, error) {
	_, err := data_store.CheckForOne(dbhandle, "cookbooks", name)
	if err == nil {
		return true, nil
	} else {
		if err != sql.ErrNoRows {
			return false, err
		} else {
			return false, nil
		}
	}
}

func (c *Cookbook) fillCookbookFromSQL(row data_store.ResRow) error {
	err := row.Scan(&c.id, &c.Name)
	if err != nil {
		return err
	}
	return nil
}

func (cbv *CookbookVersion)fillCookbookVersionFromSQL(row data_store.ResRow) error {
	var (
		defb []byte
		libb []byte
		attb []byte
		recb []byte
		prob []byte
		resb []byte
		temb []byte
		roob []byte
		filb []byte
		metb []byte
		major int64
		minor int64
		patch int64
	)
	err := row.Scan(&cbv.id, &cbv.cookbook_id, &defb, &libb, &attb, &recb, &prob, &resb, &temb, &roob, &filb, &metb, &major, &minor, &patch, &cbv.IsFrozen, &cbv.CookbookName)
	if err != nil {
		return err
	}
	/* Now... populate it. :-/ */
	// These may need to accept x.y versions with only two elements
	// instead of x.y.0 with the added default 0 patch number.
	cbv.Version = fmt.Sprintf("%d.%d.%d", major, minor, patch)
	cbv.Name = fmt.Sprintf("%s-%s", cbv.CookbookName, cbv.Version)
	cbv.ChefType = "cookbook_version"
	cbv.JsonClass = "Chef::CookbookVersion"

	/* TODO: experiment some more with getting this done with
	 * pointers. */
	err = data_store.DecodeBlob(metb, &cbv.Metadata)
	if err != nil {
		return err
	}
	err = data_store.DecodeBlob(defb, &cbv.Definitions)
	if err != nil {
		return err
	}
	err = data_store.DecodeBlob(libb, &cbv.Libraries)
	if err != nil {
		return err
	}
	err = data_store.DecodeBlob(attb, &cbv.Attributes)
	if err != nil {
		return err
	}
	err = data_store.DecodeBlob(recb, &cbv.Recipes)
	if err != nil {
		return err
	}
	err = data_store.DecodeBlob(prob, &cbv.Providers)
	if err != nil {
		return err
	}
	err = data_store.DecodeBlob(temb, &cbv.Templates)
	if err != nil {
		return err
	}
	err = data_store.DecodeBlob(resb, &cbv.Resources)
	if err != nil {
		return err
	}
	err = data_store.DecodeBlob(roob, &cbv.RootFiles)
	if err != nil {
		return err
	}
	err = data_store.DecodeBlob(filb, &cbv.Files)
	if err != nil {
		return err
	}
	data_store.ChkNilArray(cbv)

	return nil
}

func (cbv *CookbookVersion) updateCookbookVersionSQL() util.Gerror {
	// Preparing the complex data structures to be saved 
	defb, deferr := data_store.EncodeBlob(cbv.Definitions)
	if deferr != nil {
		gerr := util.Errorf(deferr.Error())
		gerr.SetStatus(http.StatusInternalServerError)
		return gerr
	}
	libb, liberr := data_store.EncodeBlob(cbv.Libraries)
	if liberr != nil {
		gerr := util.Errorf(liberr.Error())
		gerr.SetStatus(http.StatusInternalServerError)
		return gerr
	}
	attb, atterr := data_store.EncodeBlob(cbv.Attributes)
	if atterr != nil {
		gerr := util.Errorf(atterr.Error())
		gerr.SetStatus(http.StatusInternalServerError)
		return gerr
	}
	recb, recerr := data_store.EncodeBlob(cbv.Recipes)
	if recerr != nil {
		gerr := util.Errorf(recerr.Error())
		gerr.SetStatus(http.StatusInternalServerError)
		return gerr
	}
	prob, proerr := data_store.EncodeBlob(cbv.Providers)
	if proerr != nil {
		gerr := util.Errorf(proerr.Error())
		gerr.SetStatus(http.StatusInternalServerError)
		return gerr
	}
	resb, reserr := data_store.EncodeBlob(cbv.Resources)
	if reserr != nil {
		gerr := util.Errorf(reserr.Error())
		gerr.SetStatus(http.StatusInternalServerError)
		return gerr
	}
	temb, temerr := data_store.EncodeBlob(cbv.Templates)
	if temerr != nil {
		gerr := util.Errorf(temerr.Error())
		gerr.SetStatus(http.StatusInternalServerError)
		return gerr
	}
	roob, rooerr := data_store.EncodeBlob(cbv.RootFiles)
	if rooerr != nil {
		gerr := util.Errorf(rooerr.Error())
		gerr.SetStatus(http.StatusInternalServerError)
		return gerr
	}
	filb, filerr := data_store.EncodeBlob(cbv.Files)
	if filerr != nil {
		gerr := util.Errorf(filerr.Error())
		gerr.SetStatus(http.StatusInternalServerError)
		return gerr
	}
	metb, meterr := data_store.EncodeBlob(cbv.Metadata)
	if meterr != nil {
		gerr := util.Errorf(meterr.Error())
		gerr.SetStatus(http.StatusInternalServerError)
		return gerr
	}
	/* version already validated */
	maj, min, patch, _ := extractVerNums(cbv.Version)
	if config.Config.UseMySQL {
		return cbv.updateCookbookVersionMySQL(defb, libb, attb, recb, prob, resb, temb, roob, filb, metb, maj, min, patch)
	} else if config.Config.UsePostgreSQL {
		return cbv.updateCookbookVersionPostgreSQL(defb, libb, attb, recb, prob, resb, temb, roob, filb, metb, maj, min, patch) 
	}
	gerr := util.Errorf("Somehow we ended up in an impossible place trying to use an unsupported db engine")
	gerr.SetStatus(http.StatusInternalServerError)
	return gerr
}

func allCookbooksSQL() []*Cookbook {
	cookbooks := make([]*Cookbook, 0)
	var sqlStatement string
	if config.Config.UseMySQL {
		sqlStatement = "SELECT id, name FROM cookbooks"
	} else {
		sqlStatement = "SELECT id, name FROM goiardi.cookbooks"
	}
	stmt, err := data_store.Dbh.Prepare(sqlStatement)
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()
	rows, qerr := stmt.Query()
	if qerr != nil {
		if qerr == sql.ErrNoRows {
			return cookbooks
		}
		log.Fatal(qerr)
	}
	for rows.Next() {
		cb := new(Cookbook)
		err = cb.fillCookbookFromSQL(rows)
		if err != nil {
			log.Fatal(err)
		}
		cb.Versions = make(map[string]*CookbookVersion)
		cookbooks = append(cookbooks, cb)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		log.Fatal(err)
	}
	return cookbooks
}

func getCookbookSQL(name string) (*Cookbook, error) {
	cookbook := new(Cookbook)
	var sqlStatement string
	if config.Config.UseMySQL {
		sqlStatement = "SELECT id, name FROM cookbooks WHERE name = ?"
	} else if config.Config.UsePostgreSQL {
		sqlStatement = "SELECT id, name FROM goiardi.cookbooks WHERE name = $1"
	}
	stmt, err := data_store.Dbh.Prepare(sqlStatement)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	
	row := stmt.QueryRow(name)
	err = cookbook.fillCookbookFromSQL(row)
	if err != nil {
		return nil, err
	}
	cookbook.Versions = make(map[string]*CookbookVersion)

	return cookbook, nil
}

func (c *Cookbook) deleteCookbookSQL() error {
	tx, err := data_store.Dbh.Begin()
	if err != nil {
		return err
	}
	/* Delete the versions first. */
	/* First delete the hashes. This is a relatively unlikely 
	 * scenario, but it's best to make sure to reap any straggling
	 * versions and file hashes. */
	fileHashes := make([]string, 0)
	for _, cbv := range c.sortedVersions() {
		fileHashes = append(fileHashes, cbv.fileHashes()...)
	}
	sort.Strings(fileHashes)
	fileHashes = removeDupHashes(fileHashes)
	// NOTE: I had this twice for some reason. See why it's here towards the
	// beginning and not just the end -- might have been from general hash
	// deletion with mysql problems earlier.
	//c.deleteHashes(fileHashes)
	
	if config.Config.UseMySQL {
		_, err = tx.Exec("DELETE FROM cookbook_versions WHERE cookbook_id = ?", c.id)
	} else if config.Config.UsePostgreSQL {
		_, err = tx.Exec("DELETE FROM goiardi.cookbook_versions WHERE cookbook_id = $1", c.id)
	}
	
	if err != nil && err != sql.ErrNoRows {
		terr := tx.Rollback()
		if terr != nil {
			err = fmt.Errorf("deleting cookbook versions for %s had an error '%s', and then rolling back the transaction gave another error '%s'", c.Name, err.Error(), terr.Error())
		}
		return err
	}
	if config.Config.UseMySQL {
		_, err = tx.Exec("DELETE FROM cookbooks WHERE id = ?", c.id)
	} else if config.Config.UsePostgreSQL {
		_, err = tx.Exec("DELETE FROM goiardi.cookbooks WHERE id = $1", c.id)
	}
	if err != nil {
		terr := tx.Rollback()
		if terr != nil {
			err = fmt.Errorf("deleting cookbook versions for %s had an error '%s', and then rolling back the transaction gave another error '%s'", c.Name, err.Error(), terr.Error())
		}
		return err
	}
	tx.Commit()
	c.deleteHashes(fileHashes)

	return nil
}

func getCookbookListSQL() []string {
	cb_list := make([]string, 0)
	
	var sqlStatement string
	if config.Config.UseMySQL {
		sqlStatement = "SELECT name FROM cookbooks"
	} else if config.Config.UsePostgreSQL {
		sqlStatement = "SELECT name FROM goiardi.cookbooks"
	}
	rows, err := data_store.Dbh.Query(sqlStatement)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Fatal(err)
		}
		rows.Close()
		return cb_list
	}
	for rows.Next() {
		var cb_name string
		err = rows.Scan(&cb_name)
		if err != nil {
			rows.Close()
			log.Fatal(err)
		}
		cb_list = append(cb_list, cb_name)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		log.Fatal(err)
	}
	return cb_list
}

func (c *Cookbook) sortedCookbookVersionsSQL() ([]*CookbookVersion) {
	sorted := make([]*CookbookVersion, 0)
	
	var sqlStatement string
	if config.Config.UseMySQL {
		sqlStatement = "SELECT cv.id, cookbook_id, definitions, libraries, attributes, recipes, providers, resources, templates, root_files, files, metadata, major_ver, minor_ver, patch_ver, frozen, c.name FROM cookbook_versions cv LEFT JOIN cookbooks c ON cv.cookbook_id = c.id WHERE cookbook_id = ? ORDER BY major_ver DESC, minor_ver DESC, patch_ver DESC"
	} else {
		sqlStatement = "SELECT cv.id, cookbook_id, definitions, libraries, attributes, recipes, providers, resources, templates, root_files, files, metadata, major_ver, minor_ver, patch_ver, frozen, c.name FROM goiardi.cookbook_versions cv LEFT JOIN goiardi.cookbooks c ON cv.cookbook_id = c.id WHERE cookbook_id = $1 ORDER BY major_ver DESC, minor_ver DESC, patch_ver DESC"
	}
	stmt, err := data_store.Dbh.Prepare(sqlStatement)

	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()
	
	rows, qerr := stmt.Query(c.id)
	if qerr != nil {
		if qerr == sql.ErrNoRows {
			return sorted
		}
		log.Fatal(qerr)
	}
	for rows.Next() {
		cbv := new(CookbookVersion)
		err = cbv.fillCookbookVersionFromSQL(rows)
		if err != nil {
			log.Fatal(err)
		}
		// may as well populate this while we have it
		c.Versions[cbv.Version] = cbv
		sorted = append(sorted, cbv)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		log.Fatal(err)
	}
	return sorted
}

func (c *Cookbook)getCookbookVersionSQL(cbVersion string) (*CookbookVersion, error) {
	cbv := new(CookbookVersion)
	maj, min, patch, cverr := extractVerNums(cbVersion)
	if cverr != nil {
		return nil, cverr
	}
	var sqlStatement string
	if config.Config.UseMySQL {
		sqlStatement = "SELECT cv.id, cookbook_id, definitions, libraries, attributes, recipes, providers, resources, templates, root_files, files, metadata, major_ver, minor_ver, patch_ver, frozen, c.name FROM cookbook_versions cv LEFT JOIN cookbooks c ON cv.cookbook_id = c.id WHERE cookbook_id = ? AND major_ver = ? AND minor_ver = ? AND patch_ver = ?"
	} else if config.Config.UsePostgreSQL {
		sqlStatement = "SELECT cv.id, cookbook_id, definitions, libraries, attributes, recipes, providers, resources, templates, root_files, files, metadata, major_ver, minor_ver, patch_ver, frozen, c.name FROM goiardi.cookbook_versions cv LEFT JOIN goiardi.cookbooks c ON cv.cookbook_id = c.id WHERE cookbook_id = $1 AND major_ver = $2 AND minor_ver = $3 AND patch_ver = $4"
	}
	stmt, err := data_store.Dbh.Prepare(sqlStatement)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	row := stmt.QueryRow(c.id, maj, min, patch)
	err = cbv.fillCookbookVersionFromSQL(row)
	if err != nil {
		return nil, err
	} 

	return cbv, nil
}

func (cbv *CookbookVersion)deleteCookbookVersionSQL() util.Gerror {
	tx, err := data_store.Dbh.Begin()
	if err != nil {
		gerr := util.Errorf(err.Error())
		gerr.SetStatus(http.StatusInternalServerError)
		return gerr
	}

	if config.Config.UseMySQL {
		_, err = tx.Exec("DELETE FROM cookbook_versions WHERE id = ?", cbv.id)
	} else if config.Config.UsePostgreSQL {
		_, err = tx.Exec("DELETE FROM goiardi.cookbook_versions WHERE id = $1", cbv.id)
	}

	if err != nil {
		terr := tx.Rollback()
		if terr != nil {
			err = fmt.Errorf("deleting cookbook %s version %s had an error '%s', and then rolling back the transaction gave another error '%s'", cbv.CookbookName, cbv.Version, err.Error(), terr.Error())
		}
		gerr := util.Errorf(err.Error())
		gerr.SetStatus(http.StatusInternalServerError)
		return gerr
	}
	tx.Commit()
	return nil
}
