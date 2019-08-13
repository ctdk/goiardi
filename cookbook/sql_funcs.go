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

// Common SQL functions for cookbooks

package cookbook

import (
	"database/sql"
	"fmt"
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/datastore"
	"github.com/ctdk/goiardi/organization"
	"github.com/ctdk/goiardi/util"
	"log"
	"net/http"
	"regexp"
	"sort"
	"strconv"
)

func (c *Cookbook) numVersionsSQL() *int {
	cn, err := c.numVer()
	if err != nil {
		log.Fatal(err)
	}
	return &cn
}

func (c *Cookbook) numVer() (int, error) {
	var cbvCount int

	// better safe than sorry
	sqlStatement := "SELECT count(*) AS c FROM goiardi.cookbook_versions cbv JOIN goiardi.cookbooks c ON cbv.cookbook_id = c.id WHERE c.organization_id = $1 AND cbv.cookbook_id = $2"
	stmt, err := datastore.Dbh.Prepare(sqlStatement)

	if err != nil {
		return 0, err
	}
	defer stmt.Close()
	err = stmt.QueryRow(c.org.GetId(), c.id).Scan(&cbvCount)
	if err != nil {
		if err == sql.ErrNoRows {
			cbvCount = 0
		} else {
			return 0, err
		}
	}
	return cbvCount, nil
}

func checkForCookbookSQL(dbhandle datastore.Dbhandle, org *organization.Organization, name string) (bool, error) {
	_, err := datastore.CheckForOne(dbhandle, "cookbooks", org.GetId(), name)
	if err == nil {
		return true, nil
	}
	if err != sql.ErrNoRows {
		return false, err
	}
	return false, nil
}

func (c *Cookbook) fillCookbookFromSQL(row datastore.ResRow) error {
	err := row.Scan(&c.id, &c.Name)
	if err != nil {
		return err
	}
	return nil
}

func (cbv *CookbookVersion) fillCookbookVersionFromSQL(row datastore.ResRow) error {
	var (
		defb  []byte
		libb  []byte
		attb  []byte
		recb  []byte
		prob  []byte
		resb  []byte
		temb  []byte
		roob  []byte
		filb  []byte
		metb  []byte
		major int64
		minor int64
		patch int64
	)
	err := row.Scan(&cbv.id, &cbv.cookbookID, &defb, &libb, &attb, &recb, &prob, &resb, &temb, &roob, &filb, &metb, &major, &minor, &patch, &cbv.IsFrozen, &cbv.CookbookName)
	if err != nil {
		return err
	}
	/* Now... populate it. :-/ */
	// These may need to accept x.y versions with only two elements
	// instead of x.y.0 with the added default 0 patch number.
	cbv.Version = fmt.Sprintf("%d.%d.%d", major, minor, patch)
	cbv.Name = fmt.Sprintf("%s-%s", cbv.CookbookName, cbv.Version)
	cbv.ChefType = "cookbook_version"
	cbv.JSONClass = "Chef::CookbookVersion"

	/* TODO: experiment some more with getting this done with
	 * pointers. */
	err = datastore.DecodeBlob(metb, &cbv.Metadata)
	if err != nil {
		return err
	}
	err = datastore.DecodeBlob(defb, &cbv.Definitions)
	if err != nil {
		return err
	}
	err = datastore.DecodeBlob(libb, &cbv.Libraries)
	if err != nil {
		return err
	}
	err = datastore.DecodeBlob(attb, &cbv.Attributes)
	if err != nil {
		return err
	}
	err = datastore.DecodeBlob(recb, &cbv.Recipes)
	if err != nil {
		return err
	}
	err = datastore.DecodeBlob(prob, &cbv.Providers)
	if err != nil {
		return err
	}
	err = datastore.DecodeBlob(temb, &cbv.Templates)
	if err != nil {
		return err
	}
	err = datastore.DecodeBlob(resb, &cbv.Resources)
	if err != nil {
		return err
	}
	err = datastore.DecodeBlob(roob, &cbv.RootFiles)
	if err != nil {
		return err
	}
	err = datastore.DecodeBlob(filb, &cbv.Files)
	if err != nil {
		return err
	}
	datastore.ChkNilArray(cbv)

	return nil
}

func (cbv *CookbookVersion) updateCookbookVersionSQL() util.Gerror {
	// Preparing the complex data structures to be saved
	defb, deferr := datastore.EncodeBlob(cbv.Definitions)
	if deferr != nil {
		gerr := util.Errorf(deferr.Error())
		gerr.SetStatus(http.StatusInternalServerError)
		return gerr
	}
	libb, liberr := datastore.EncodeBlob(cbv.Libraries)
	if liberr != nil {
		gerr := util.Errorf(liberr.Error())
		gerr.SetStatus(http.StatusInternalServerError)
		return gerr
	}
	attb, atterr := datastore.EncodeBlob(cbv.Attributes)
	if atterr != nil {
		gerr := util.Errorf(atterr.Error())
		gerr.SetStatus(http.StatusInternalServerError)
		return gerr
	}
	recb, recerr := datastore.EncodeBlob(cbv.Recipes)
	if recerr != nil {
		gerr := util.Errorf(recerr.Error())
		gerr.SetStatus(http.StatusInternalServerError)
		return gerr
	}
	prob, proerr := datastore.EncodeBlob(cbv.Providers)
	if proerr != nil {
		gerr := util.Errorf(proerr.Error())
		gerr.SetStatus(http.StatusInternalServerError)
		return gerr
	}
	resb, reserr := datastore.EncodeBlob(cbv.Resources)
	if reserr != nil {
		gerr := util.Errorf(reserr.Error())
		gerr.SetStatus(http.StatusInternalServerError)
		return gerr
	}
	temb, temerr := datastore.EncodeBlob(cbv.Templates)
	if temerr != nil {
		gerr := util.Errorf(temerr.Error())
		gerr.SetStatus(http.StatusInternalServerError)
		return gerr
	}
	roob, rooerr := datastore.EncodeBlob(cbv.RootFiles)
	if rooerr != nil {
		gerr := util.Errorf(rooerr.Error())
		gerr.SetStatus(http.StatusInternalServerError)
		return gerr
	}
	filb, filerr := datastore.EncodeBlob(cbv.Files)
	if filerr != nil {
		gerr := util.Errorf(filerr.Error())
		gerr.SetStatus(http.StatusInternalServerError)
		return gerr
	}
	metb, meterr := datastore.EncodeBlob(cbv.Metadata)
	if meterr != nil {
		gerr := util.Errorf(meterr.Error())
		gerr.SetStatus(http.StatusInternalServerError)
		return gerr
	}

	/* version already validated */
	maj, min, patch, _ := extractVerNums(cbv.Version)

	return cbv.updateCookbookVersionPostgreSQL(defb, libb, attb, recb, prob, resb, temb, roob, filb, metb, maj, min, patch)
}

func allCookbooksSQL(org *organization.Organization) []*Cookbook {
	var cookbooks []*Cookbook
	sqlStatement := "SELECT id, name FROM goiardi.cookbooks WHERE organization_id = $1"
	stmt, err := datastore.Dbh.Prepare(sqlStatement)
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	rows, qerr := stmt.Query(org.GetId())
	if qerr != nil {
		if qerr == sql.ErrNoRows {
			return cookbooks
		}
		log.Fatal(qerr)
	}

	for rows.Next() {
		cb := new(Cookbook)
		cb.org = org
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

func getCookbookSQL(name string, org *organization.Organization) (*Cookbook, error) {
	cookbook := new(Cookbook)
	cookbook.org = org

	sqlStatement := "SELECT id, name FROM goiardi.cookbooks WHERE organization_id = $1 AND name = $2"
	stmt, err := datastore.Dbh.Prepare(sqlStatement)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	row := stmt.QueryRow(org.GetId(), name)
	err = cookbook.fillCookbookFromSQL(row)
	if err != nil {
		return nil, err
	}
	cookbook.Versions = make(map[string]*CookbookVersion)

	return cookbook, nil
}

func (c *Cookbook) deleteCookbookSQL() error {
	tx, err := datastore.Dbh.Begin()
	if err != nil {
		return err
	}
	/* Delete the versions first. */
	/* First delete the hashes. This is a relatively unlikely
	 * scenario, but it's best to make sure to reap any straggling
	 * versions and file hashes. */
	var fileHashes []string
	for _, cbv := range c.sortedVersions() {
		fileHashes = append(fileHashes, cbv.fileHashes()...)
	}
	sort.Strings(fileHashes)
	fileHashes = removeDupHashes(fileHashes)

	_, err = tx.Exec("DELETE FROM goiardi.cookbook_versions WHERE cookbook_id = $1", c.id)

	if err != nil && err != sql.ErrNoRows {
		terr := tx.Rollback()
		if terr != nil {
			err = fmt.Errorf("deleting cookbook versions for %s had an error '%s', and then rolling back the transaction gave another error '%s'", c.Name, err.Error(), terr.Error())
		}
		return err
	}

	_, err = tx.Exec("DELETE FROM goiardi.cookbooks WHERE id = $1", c.id)
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

func getCookbookListSQL(org *organization.Organization) []string {
	var cbList []string

	// what on earth did I have this using datastore.Dbh.Query before?
	sqlStatement := "SELECT name FROM goiardi.cookbooks WHERE organization_id = $1"
	stmt, err := datastore.Dbh.Prepare(sqlStatement)
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	rows, err := stmt.Query(org.GetId())
	if err != nil {
		if err != sql.ErrNoRows {
			log.Fatal(err)
		}
		rows.Close()
		return cbList
	}
	for rows.Next() {
		var cbName string
		err = rows.Scan(&cbName)
		if err != nil {
			rows.Close()
			log.Fatal(err)
		}
		cbList = append(cbList, cbName)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		log.Fatal(err)
	}
	return cbList
}

func (c *Cookbook) sortedCookbookVersionsSQL() []*CookbookVersion {
	var sorted []*CookbookVersion

	sqlStatement := "SELECT cv.id, cookbook_id, definitions, libraries, attributes, recipes, providers, resources, templates, root_files, files, metadata, major_ver, minor_ver, patch_ver, frozen, c.name FROM goiardi.cookbook_versions cv LEFT JOIN goiardi.cookbooks c ON cv.cookbook_id = c.id WHERE cookbook_id = $1 ORDER BY major_ver DESC, minor_ver DESC, patch_ver DESC"

	stmt, err := datastore.Dbh.Prepare(sqlStatement)
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
		cbv.org = c.org
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

func (c *Cookbook) getCookbookVersionSQL(cbVersion string) (*CookbookVersion, error) {
	cbv := new(CookbookVersion)
	maj, min, patch, cverr := extractVerNums(cbVersion)
	if cverr != nil {
		return nil, cverr
	}
	sqlStatement := "SELECT cv.id, cookbook_id, definitions, libraries, attributes, recipes, providers, resources, templates, root_files, files, metadata, major_ver, minor_ver, patch_ver, frozen, c.name FROM goiardi.cookbook_versions cv LEFT JOIN goiardi.cookbooks c ON cv.cookbook_id = c.id WHERE cookbook_id = $1 AND major_ver = $2 AND minor_ver = $3 AND patch_ver = $4"
	stmt, err := datastore.Dbh.Prepare(sqlStatement)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	cbv.org = c.org
	row := stmt.QueryRow(c.id, maj, min, patch)
	err = cbv.fillCookbookVersionFromSQL(row)
	if err != nil {
		return nil, err
	}

	return cbv, nil
}

func (c *Cookbook) checkCookbookVersionSQL(cbVersion string) (bool, error) {
	var found bool

	if cbVersion == "_latest" {
		cn, err := c.numVer()
		if err != nil {
			return false, err
		}
		if cn != 0 {
			found = true
		}
		return false, nil
	}

	maj, min, patch, cverr := extractVerNums(cbVersion)
	if cverr != nil {
		return false, cverr
	}

	sqlStatement := "SELECT COUNT(cv.id) FROM goiardi.cookbook_versions cv LEFT JOIN goiardi.cookbooks c ON cv.cookbook_id = c.id WHERE cookbook_id = $1 AND major_ver = $2 AND minor_ver = $3 AND patch_ver = $4"

	stmt, err := datastore.Dbh.Prepare(sqlStatement)
	if err != nil {
		return false, err
	}
	defer stmt.Close()

	var cn int
	err = stmt.QueryRow(c.id, maj, min, patch).Scan(&c)
	if err != nil && err != sql.ErrNoRows {
		return false, err
	}
	if cn != 0 {
		found = true
	}

	return found, nil
}

func (cbv *CookbookVersion) deleteCookbookVersionSQL() util.Gerror {
	tx, err := datastore.Dbh.Begin()
	if err != nil {
		gerr := util.Errorf(err.Error())
		gerr.SetStatus(http.StatusInternalServerError)
		return gerr
	}

	_, err = tx.Exec("DELETE FROM goiardi.cookbook_versions WHERE id = $1", cbv.id)

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

func universeSQL(org *organization.Organization) map[string]map[string]interface{} {
	universe := make(map[string]map[string]interface{})
	var (
		major int64
		minor int64
		patch int64
	)
	var name string

	sqlStatement := "SELECT major_ver, minor_ver, patch_ver, c.name, metadata->>'dependencies' FROM goiardi.cookbook_versions cv LEFT JOIN goiardi.cookbooks c ON cv.cookbook_id = c.id ORDER BY cv.cookbook_id, major_ver DESC, minor_ver DESC, patch_ver DESC WHERE c.organization_id = $1"
	stmt, err := datastore.Dbh.Prepare(sqlStatement)

	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	rows, qerr := stmt.Query(org.GetId())
	if qerr != nil {
		if qerr == sql.ErrNoRows {
			return universe
		}
		log.Fatal(qerr)
	}

	for rows.Next() {
		var metb sql.RawBytes
		metadata := make(map[string]interface{})
		u := make(map[string]interface{})
		err := rows.Scan(&major, &minor, &patch, &name, &metb)
		if err != nil {
			log.Fatal(err)
		}
		err = datastore.DecodeBlob(metb, &metadata)
		if err != nil {
			log.Fatal(err)
		}
		version := fmt.Sprintf("%d.%d.%d", major, minor, patch)
		customURL := fmt.Sprintf("%s/cookbook/%s/%s", org.OrgURLBase(), name, version)
		u["location_path"] = util.CustomURL(customURL)
		u["location_type"] = "chef_server"

		if config.Config.UsePostgreSQL {
			u["dependencies"] = metadata
		} else {
			u["dependencies"] = metadata["dependencies"]
		}
		if _, ok := universe[name]; !ok {
			universe[name] = make(map[string]interface{})
		}
		universe[name][version] = u
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		log.Fatal(err)
	}
	return universe
}

func cookbookListerSQL(numResults interface{}, org *organization.Organization) map[string]interface{} {
	var numVersions int
	allVersions := false

	cl := make(map[string]interface{})

	if numResults != "" && numResults != "all" {
		numVersions, _ = strconv.Atoi(numResults.(string))
	} else if numResults == "" {
		numVersions = 1
	} else {
		allVersions = true
	}

	sqlStatement := "SELECT version, name FROM goiardi.joined_cookbook_version WHERE organization_id = $1 ORDER BY name, major_ver desc, minor_ver desc, patch_ver DESC"
	stmt, err := datastore.Dbh.Prepare(sqlStatement)

	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	rows, qerr := stmt.Query(org.GetId())
	if qerr != nil {
		if qerr == sql.ErrNoRows {
			return cl
		}
		log.Fatal(qerr)
	}
	scratch := make(map[string][]string)
	for rows.Next() {
		var n, v string
		err := rows.Scan(&v, &n)
		if err != nil {
			log.Fatal(err)
		}
		scratch[n] = append(scratch[n], v)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		log.Fatal(err)
	}
	for name, versions := range scratch {
		nr := 0
		cburl := fmt.Sprintf("%s/cookbooks/%s", org.OrgURLBase(), name)
		cb := make(map[string]interface{})
		cb["url"] = util.CustomURL(cburl)
		cb["versions"] = make([]interface{}, 0)
		for _, ver := range versions {
			if !allVersions && nr >= numVersions {
				break
			}
			cv := make(map[string]string)
			cv["url"] = util.CustomURL(fmt.Sprintf("%s/cookbooks/%s/%s", org.OrgURLBase(), name, ver))
			cv["version"] = ver
			cb["versions"] = append(cb["versions"].([]interface{}), cv)
			nr++
		}
		cl[name] = cb
	}
	return cl
}

func cookbookRecipesSQL(org *organization.Organization) ([]string, util.Gerror) {
	sqlStatement := "SELECT version, name, recipes FROM goiardi.joined_cookbook_version WHERE organization_id = $1 ORDER BY name, major_ver desc, minor_ver desc, patch_ver desc"
	stmt, err := datastore.Dbh.Prepare(sqlStatement)

	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	rlist := make([]string, 0)

	rows, qerr := stmt.Query(org.GetId())
	if qerr != nil {
		if qerr == sql.ErrNoRows {
			return rlist, nil
		}
		return nil, util.CastErr(qerr)
	}
	seen := make(map[string]bool)
	for rows.Next() {
		var n, v string
		var rec sql.RawBytes
		recipes := make([]map[string]interface{}, 0)
		err := rows.Scan(&v, &n, &rec)
		if seen[n] {
			continue
		}
		if err != nil {
			return nil, util.CastErr(err)
		}
		err = datastore.DecodeBlob(rec, &recipes)
		if err != nil {
			return nil, util.CastErr(err)
		}
		rltmp := make([]string, len(recipes))
		ci := 0
		for _, r := range recipes {
			rm := regexp.MustCompile(`(.*?)\.rb`)
			rfind := rm.FindStringSubmatch(r["name"].(string))
			if rfind == nil {
				/* unlikely */
				err := util.Errorf("No recipe name found")
				return nil, err
			}
			rbase := rfind[1]
			var rname string
			if rbase == "default" {
				rname = n
			} else {
				rname = fmt.Sprintf("%s::%s", n, rbase)
			}
			rltmp[ci] = rname
			ci++
		}
		rlist = append(rlist, rltmp...)
		seen[n] = true
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		log.Fatal(err)
	}
	sort.Strings(rlist)
	return rlist, nil
}
