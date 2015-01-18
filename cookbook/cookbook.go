/* Cookbooks! The ultimate building block of any chef run. */

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

// Package cookbook handles the basic building block of any chef (or goiardi)
// run, the humble cookbook.
package cookbook

import (
	"database/sql"
	"fmt"
	"github.com/ctdk/goas/v2/logger"
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/datastore"
	"github.com/ctdk/goiardi/filestore"
	"github.com/ctdk/goiardi/util"
	"github.com/hashicorp/terraform/depgraph"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// VersionStrings is a type to make version strings with the format "x.y.z"
// sortable.
type VersionStrings []string

// The Cookbook struct holds an array of cookbook versions, which is where the
// run lists, definitions, attributes, etc. are.
type Cookbook struct {
	Name        string
	Versions    map[string]*CookbookVersion
	latest      *CookbookVersion
	numVersions *int
	id          int32
}

/* We... want the JSON tags for this. */

// CookbookVersion is the meat of the cookbook. This is what's set when a new
// cookbook is uploaded.
type CookbookVersion struct {
	CookbookName string                   `json:"cookbook_name"`
	Name         string                   `json:"name"`
	Version      string                   `json:"version"`
	ChefType     string                   `json:"chef_type"`
	JSONClass    string                   `json:"json_class"`
	Definitions  []map[string]interface{} `json:"definitions"`
	Libraries    []map[string]interface{} `json:"libraries"`
	Attributes   []map[string]interface{} `json:"attributes"`
	Recipes      []map[string]interface{} `json:"recipes"`
	Providers    []map[string]interface{} `json:"providers"`
	Resources    []map[string]interface{} `json:"resources"`
	Templates    []map[string]interface{} `json:"templates"`
	RootFiles    []map[string]interface{} `json:"root_files"`
	Files        []map[string]interface{} `json:"files"`
	IsFrozen     bool                     `json:"frozen?"`
	Metadata     map[string]interface{}   `json:"metadata"`
	id           int32
	cookbookID   int32
}

/* Cookbook methods and functions */

// GetName returns the name of the cookbook.
func (c *Cookbook) GetName() string {
	return c.Name
}

// URLType returns the first path element in a cookbook's URL.
func (c *Cookbook) URLType() string {
	return "cookbooks"
}

// GetName returns the name of the cookbook version.
func (cbv *CookbookVersion) GetName() string {
	return cbv.Name
}

// URLType returns the first path element in a cookbook version's URL.
func (cbv *CookbookVersion) URLType() string {
	return "cookbooks"
}

// New creates a new cookbook.
func New(name string) (*Cookbook, util.Gerror) {
	var found bool
	if !util.ValidateEnvName(name) {
		err := util.Errorf("Invalid cookbook name '%s' using regex: 'Malformed cookbook name. Must only contain A-Z, a-z, 0-9, _ or -'.", name)
		return nil, err
	}
	if config.UsingDB() {
		var cerr error
		found, cerr = checkForCookbookSQL(datastore.Dbh, name)
		if cerr != nil {
			err := util.CastErr(cerr)
			err.SetStatus(http.StatusInternalServerError)
			return nil, err
		}
	} else {
		ds := datastore.New()
		_, found = ds.Get("cookbook", name)
	}
	if found {
		err := util.Errorf("Cookbook %s already exists", name)
		err.SetStatus(http.StatusConflict)
	}
	cookbook := &Cookbook{
		Name:     name,
		Versions: make(map[string]*CookbookVersion),
	}
	return cookbook, nil
}

// NumVersions returns the number of versions this cookbook has.
func (c *Cookbook) NumVersions() int {
	if config.UsingDB() {
		if c.numVersions == nil {
			c.numVersions = c.numVersionsSQL()
		}
		return *c.numVersions
	}
	return len(c.Versions)
}

// AllCookbooks returns all the cookbooks that have been uploaded to this server.
func AllCookbooks() (cookbooks []*Cookbook) {
	if config.UsingDB() {
		cookbooks = allCookbooksSQL()
		for _, c := range cookbooks {
			// populate the versions hash
			c.sortedVersions()
		}
	} else {
		cookbookList := GetList()
		for _, c := range cookbookList {
			cb, err := Get(c)
			if err != nil {
				logger.Debugf("Curious. Cookbook %s was in the cookbook list, but wasn't found when fetched. Continuing.", c)
				continue
			}
			cookbooks = append(cookbooks, cb)
		}
	}
	return cookbooks
}

// Get a cookbook.
func Get(name string) (*Cookbook, util.Gerror) {
	var cookbook *Cookbook
	var found bool
	if config.UsingDB() {
		var err error
		cookbook, err = getCookbookSQL(name)
		if err != nil {
			if err == sql.ErrNoRows {
				found = false
			} else {
				gerr := util.CastErr(err)
				gerr.SetStatus(http.StatusInternalServerError)
				return nil, gerr
			}
		} else {
			found = true
		}
	} else {
		ds := datastore.New()
		var c interface{}
		c, found = ds.Get("cookbook", name)
		if c != nil {
			cookbook = c.(*Cookbook)
		}
		/* hrm. */
		if cookbook != nil && config.Config.UseUnsafeMemStore {
			for _, v := range cookbook.Versions {
				datastore.ChkNilArray(v)
			}
		}
	}
	if !found {
		err := util.Errorf("Cannot find a cookbook named %s", name)
		err.SetStatus(http.StatusNotFound)
		return nil, err
	}
	return cookbook, nil
}

// Save a cookbook to the in-memory data store or database.
func (c *Cookbook) Save() error {
	var err error
	if config.Config.UseMySQL {
		err = c.saveCookbookMySQL()
	} else if config.Config.UsePostgreSQL {
		err = c.saveCookbookPostgreSQL()
	} else {
		ds := datastore.New()
		ds.Set("cookbook", c.Name, c)
	}
	if err != nil {
		return err
	}
	return nil
}

// Delete a coookbook.
func (c *Cookbook) Delete() error {
	var err error
	if config.UsingDB() {
		err = c.deleteCookbookSQL()
	} else {
		ds := datastore.New()
		ds.Delete("cookbook", c.Name)
	}
	if err != nil {
		return err
	}
	return nil
}

// GetList gets a list of all cookbooks on this server.
func GetList() []string {
	if config.UsingDB() {
		return getCookbookListSQL()
	}
	ds := datastore.New()
	cbList := ds.GetList("cookbook")
	return cbList
}

/* Returns a sorted list of all the versions of this cookbook */
func (c *Cookbook) sortedVersions() []*CookbookVersion {
	if config.UsingDB() {
		return c.sortedCookbookVersionsSQL()
	}
	sorted := make([]*CookbookVersion, len(c.Versions))
	keys := make(VersionStrings, len(c.Versions))

	u := 0
	for k, cbv := range c.Versions {
		keys[u] = k
		u++
		datastore.ChkNilArray(cbv)
	}
	sort.Sort(sort.Reverse(keys))

	/* populate sorted now */
	for i, s := range keys {
		/* This shouldn't be able to happen, but somehow it... does? */
		if i >= len(sorted) {
			break
		}
		sorted[i] = c.Versions[s]
	}
	return sorted
}

// UpdateLatestVersion updates what the cookbook stores as the latest version
// available.
func (c *Cookbook) UpdateLatestVersion() {
	c.latest = nil
	c.LatestVersion()
}

// LatestVersion gets the latest version of this cookbook.
func (c *Cookbook) LatestVersion() *CookbookVersion {
	if c.latest == nil {
		sorted := c.sortedVersions()
		c.latest = sorted[0]
		if c.latest != nil {
			datastore.ChkNilArray(c.latest)
		}
	}
	return c.latest
}

// CookbookLister lists all of the cookbooks on the server, along with some
// information like URL, available versions, etc.
func CookbookLister(numResults interface{}) map[string]interface{} {
	if config.UsingDB() {
		return cookbookListerSQL(numResults)
	}
	cr := make(map[string]interface{})
	for _, cb := range AllCookbooks() {
		cr[cb.Name] = cb.InfoHash(numResults)
	}
	return cr
}

// CookbookLatest returns the URL of the latest version of each cookbook on the
// server.
func CookbookLatest() map[string]interface{} {
	latest := make(map[string]interface{})
	if config.UsingDB() {
		cs := CookbookLister("")
		for name, cbdata := range cs {
			if len(cbdata.(map[string]interface{})["versions"].([]interface{})) > 0 {
				latest[name] = cbdata.(map[string]interface{})["versions"].([]interface{})[0].(map[string]string)["url"]
			}
		}
	} else {
		for _, cb := range AllCookbooks() {
			latest[cb.Name] = util.CustomObjURL(cb, cb.LatestVersion().Version)
		}
	}
	return latest
}

// CookbookRecipes returns a list of all the recipes on the server in the latest
// version of each cookbook.
func CookbookRecipes() ([]string, util.Gerror) {
	if config.UsingDB() {
		return cookbookRecipesSQL()
	}
	rlist := make([]string, 0)
	for _, cb := range AllCookbooks() {
		/* Damn it, this sends back an array of
		 * all the recipes. Fill it in, and send
		 * back the JSON ourselves. */
		rlistTmp, err := cb.LatestVersion().RecipeList()
		if err != nil {
			return nil, err
		}
		rlist = append(rlist, rlistTmp...)
	}
	sort.Strings(rlist)
	return rlist, nil
}

// InfoHash gets numResults (or all if numResults is nil) versions of a
// cookbook,returning a hash describing the cookbook and the versions returned.
func (c *Cookbook) InfoHash(numResults interface{}) map[string]interface{} {
	return c.infoHashBase(numResults, "")
}

// ConstrainedInfoHash gets numResults (or all if numResults is nil) versions of
// a cookbook that match the given constraint and returns a hash describing the
// cookbook and the versions returned.
func (c *Cookbook) ConstrainedInfoHash(numResults interface{}, constraint string) map[string]interface{} {
	return c.infoHashBase(numResults, constraint)
}

// DependsCookbooks will, for the given run list and environment constraints,
// return the cookbook dependencies.
func DependsCookbooks(runList []string, envConstraints map[string]string) (map[string]interface{}, error) {
	cdList := make(map[string][]string, len(runList))
	runListRef := make([]string, len(runList))

	for i, cbV := range runList {
		var cbName string
		var constraint string
		cx := strings.Split(cbV, "@")
		cbName = strings.Split(cx[0], "::")[0]
		if len(cx) == 2 {
			constraint = fmt.Sprintf("= %s", cx[1])
		}
		cdList[cbName] = []string{constraint}
		/* There's a method to our madness. We need to modify the
		 * cdList as we go along, but want the base list to remain the
		 * same. Thus, we make an additional array of cookbook names to
		 * range through. */
		runListRef[i] = cbName
	}

	for k, ec := range envConstraints {
		if _, found := cdList[k]; !found {
			continue
		} else {
			/* Check if there's a constraint in the uploaded run
			 * list. If not take the env one and move on. */
			if cdList[k][0] == "" {
				cdList[k] = []string{ec}
				continue
			}
			/* Override if the environment is more restrictive */
			_, orgver, _ := splitConstraint(cdList[k][0])
			newop, newver, nerr := splitConstraint(ec)
			if nerr != nil {
				return nil, nerr
			}
			/* if the versions are equal, take the env one */
			if orgver == newver {
				cdList[k] = []string{ec}
				continue /* and break out of this step */
			}
			/*
			switch newop {
			case ">", ">=":
				if versionLess(orgver, newver) {
					cdList[k] = []string{ec}
				}
			case "<", "<=":
				if !versionLess(orgver, newver) {
					cdList[k] = []string{ec}
				}
			case "=":
				if orgver != newver {
					err := fmt.Errorf("This run list has a constraint '%s' for %s that conflicts with '%s' in the environment's cookbook versions.", cdList[k][0], k, ec)
					return nil, err
				}
			case "~>":
				if action := verConstraintCheck(orgver, newver, newop); action == "ok" {
					cdList[k] = []string{ec}
				} else {
					err := fmt.Errorf("This run list has a constraint '%s' for %s that conflicts with '%s' in the environment's cookbook versions.", cdList[k][0], k, ec)
					return nil, err
				}
			default:
				err := fmt.Errorf("An unlikely occurance, but the constraint '%s' for cookbook %s in this environment is impossible.", ec, k)
				return nil, err
			}
			*/
			// Try just appending the dependency and resolve the
			// graph later
			cdList[k] = append(cdList[k], ec)
		}
	}

	/* Build a slice holding all the needed cookbooks. */
	for _, cbName := range runListRef {
		c, err := Get(cbName)
		if err != nil {
			return nil, err
		}
		cbv := c.LatestConstrained(cdList[cbName][0])
		if cbv == nil {
			return nil, fmt.Errorf("No cookbook found for %s that satisfies constraint '%s'", c.Name, cdList[cbName][0])
		}

		nerr := cbv.resolveDependencies(cdList)
		if nerr != nil {
			return nil, nerr
		}
	}

	cookbookDeps := make(map[string]interface{}, len(cdList))
	for cname, traints := range cdList {
		cb, err := Get(cname)
		/* Although we would have already seen this, but being careful
		 * rarely hurt. */
		if err != nil {
			return nil, err
		}
		var gcbv *CookbookVersion

		for _, cv := range cb.sortedVersions() {
		Vers:
			for _, ct := range traints {
				if ct != "" { // no constraint
					op, ver, err := splitConstraint(ct)
					if err != nil {
						return nil, err
					}
					if action := verConstraintCheck(cv.Version, ver, op); action != "ok" {
						// BREAK THIS LOOP, BUT CONTINUE THE cv LOOP. HMM
						continue Vers
					}
				}
			}
			/* If we pass the constraint tests, set gcbv to cv and
			 * break. */
			gcbv = cv
			break
		}
		if gcbv == nil {
			err := fmt.Errorf("Unfortunately no version of %s could satisfy the requested constraints: %s", cname, strings.Join(traints, ", "))
			return nil, err
		}
		gcbvJSON := gcbv.ToJSON("POST")
		/* Sigh. For some reason, *some* places want nothing
		 * sent for cookbook information divisions like
		 * attributes, libraries, providers, etc. However,
		 * others will flip out if nothing is sent at all, and
		 * environment/<env>/cookbook_versions is one of them.
		 * Go through the list of possibly guilty divisions and
		 * set them to an empty slice of maps if they're nil. */
		chkDiv := []string{"definitions", "libraries", "attributes", "providers", "resources", "templates", "root_files", "files"}
		for _, cd := range chkDiv {
			if gcbvJSON[cd] == nil {
				gcbvJSON[cd] = make([]map[string]interface{}, 0)
			}
		}
		cookbookDeps[gcbv.CookbookName] = gcbvJSON
	}

	return cookbookDeps, nil
}

func (cbv *CookbookVersion) resolveDependencies(cdList map[string][]string) error {
	depList := cbv.Metadata["dependencies"].(map[string]interface{})

	for r, c2 := range depList {
		c := c2.(string)
		depCb, err := Get(r)
		if err != nil {
			return err
		}
		debCbv := depCb.LatestConstrained(c)
		if debCbv == nil {
			err := fmt.Errorf("No cookbook version for %s satisfies constraint '%s'.", r, c)
			return err
		}

		/* Do we satisfy the constraints we have? */
		if constraints, found := cdList[r]; found {
			for _, dcon := range constraints {
				if dcon != "" {
					op, ver, err := splitConstraint(dcon)
					if err != nil {
						return err
					}
					stat := verConstraintCheck(debCbv.Version, ver, op)
					if stat != "ok" {
						err := fmt.Errorf("Oh no! Cookbook %s (ver %s) depends on a version of cookbook %s matching the constraint '%s', but that constraint conflicts with the previous constraint of '%s'. Bailing, sorry.", cbv.CookbookName, cbv.Version, debCbv.CookbookName, c, dcon)
						return err
					}
				}
			}
		} else {
			/* Add our constraint */
			cdList[r] = []string{c}
		}

		nerr := debCbv.resolveDependencies(cdList)
		if nerr != nil {
			return nerr
		}
	}
	return nil
}

func splitConstraint(constraint string) (string, string, error) {
	t1 := strings.Split(constraint, " ")
	if len(t1) != 2 {
		err := fmt.Errorf("Constraint '%s' was not well-formed.", constraint)
		return "", "", err
	}
	op := t1[0]
	ver := t1[1]
	return op, ver, nil
}

func (c *Cookbook) infoHashBase(numResults interface{}, constraint string) map[string]interface{} {
	cbHash := make(map[string]interface{})
	cbHash["url"] = util.ObjURL(c)

	nr := 0

	/* Working to maintain Chef server behavior here. We need to make "all"
	 * give all versions of the cookbook and make no value give one version,
	 * but keep 0 as invalid input that gives zero results back. This might
	 * be an area worth breaking. */
	var numVersions int
	allVersions := false

	if numResults != "" && numResults != "all" {
		numVersions, _ = strconv.Atoi(numResults.(string))
	} else if numResults == "" {
		numVersions = 1
	} else {
		allVersions = true
	}

	cbHash["versions"] = make([]interface{}, 0)

	var constraintVersion string
	var constraintOp string
	if constraint != "" {
		traints := strings.Split(constraint, " ")
		/* If the constraint isn't well formed like ">= 1.2.3", log the
		 * fact and ignore the constraint. */
		if len(traints) == 2 {
			constraintVersion = traints[1]
			constraintOp = traints[0]
		} else {
			logger.Warningf("Constraint '%s' for cookbook %s was badly formed -- bailing.\n", constraint, c.Name)
			return nil
		}
	}

VerLoop:
	for _, cv := range c.sortedVersions() {
		if !allVersions && nr >= numVersions {
			break
		}
		/* Version constraint checking. */
		if constraint != "" {
			conAction := verConstraintCheck(cv.Version, constraintVersion, constraintOp)
			switch conAction {
			case "skip":
				/* Skip this version, keep going. */
				continue VerLoop
			case "break":
				/* Stop processing entirely. */
				break VerLoop
				/* Default action is, of course, to continue on
				 * like nothing happened. Later, we need to
				 * panic over an invalid constraint. */
			}
		}
		cvInfo := make(map[string]string)
		cvInfo["url"] = util.CustomObjURL(c, cv.Version)
		cvInfo["version"] = cv.Version
		cbHash["versions"] = append(cbHash["versions"].([]interface{}), cvInfo)
		nr++
	}
	return cbHash
}

// LatestConstrained returns the latest version of a cookbook that matches the
// given constraint. If no constraint is given, returns the latest version.
func (c *Cookbook) LatestConstrained(constraint string) *CookbookVersion {
	if constraint == "" {
		return c.LatestVersion()
	}
	var constraintVersion string
	var constraintOp string
	traints := strings.Split(constraint, " ")
	if len(traints) == 2 {
		constraintVersion = traints[1]
		constraintOp = traints[0]
	} else {
		logger.Warningf("Constraint '%s' for cookbook %s (in LatestConstrained) was malformed. Bailing.\n", constraint, c.Name)
		return nil
	}
	for _, cv := range c.sortedVersions() {
		action := verConstraintCheck(cv.Version, constraintVersion, constraintOp)
		/* We only want the latest that works. */
		if action == "ok" {
			return cv
		}
	}
	/* if nothing satisfied the constraint, we have to return nil */
	return nil
}

// Universe returns a hash of the cookbooks stored on this server, with a list
// of each version of each cookbook formatted to be compatible with the
// supermarket/berks /universe endpoint.
func Universe() map[string]map[string]interface{} {
	if config.UsingDB() {
		return universeSQL()
	}
	universe := make(map[string]map[string]interface{})

	for _, cb := range AllCookbooks() {
		universe[cb.Name] = cb.universeFormat()
	}
	return universe
}

// universeFormat returns a sorted list of this cookbook's versions, formatted
// to be compatible with the supermarket/berks /universe endpoint.
func (c *Cookbook) universeFormat() map[string]interface{} {
	u := make(map[string]interface{})
	for _, cbv := range c.sortedVersions() {
		v := make(map[string]interface{})
		v["location_path"] = util.CustomObjURL(c, cbv.Version)
		v["location_type"] = "chef_server"
		v["dependencies"] = cbv.Metadata["dependencies"]
		u[cbv.Version] = v
	}
	return u
}

/* CookbookVersion methods and functions */

// NewVersion creates a new version of the cookbook.
func (c *Cookbook) NewVersion(cbVersion string, cbvData map[string]interface{}) (*CookbookVersion, util.Gerror) {
	if _, err := c.GetVersion(cbVersion); err == nil {
		err := util.Errorf("Version %s of cookbook %s already exists, and shouldn't be created like this. Use UpdateVersion instead.", cbVersion, c.Name)
		err.SetStatus(http.StatusConflict)
		return nil, err
	}
	cbv := &CookbookVersion{
		CookbookName: c.Name,
		Version:      cbVersion,
		Name:         fmt.Sprintf("%s-%s", c.Name, cbVersion),
		ChefType:     "cookbook_version",
		JSONClass:    "Chef::CookbookVersion",
		IsFrozen:     false,
		cookbookID:   c.id, // should be ok even with in-mem
	}
	err := cbv.UpdateVersion(cbvData, "")
	if err != nil {
		return nil, err
	}
	/* And, dur, add it to the versions */
	c.Versions[cbVersion] = cbv

	c.numVersions = nil
	c.UpdateLatestVersion()
	c.Save()
	return cbv, nil
}

// GetVersion gets a particular version of the cookbook.
func (c *Cookbook) GetVersion(cbVersion string) (*CookbookVersion, util.Gerror) {
	if cbVersion == "_latest" {
		return c.LatestVersion(), nil
	}
	var cbv *CookbookVersion
	var found bool

	if config.UsingDB() {
		// Ridiculously cacheable, but let's get it working first. This
		// applies all over the place w/ the SQL bits.
		if cbv, found = c.Versions[cbVersion]; !found {
			var err error
			cbv, err = c.getCookbookVersionSQL(cbVersion)
			if err != nil {
				if err == sql.ErrNoRows {
					found = false
				} else {
					gerr := util.Errorf(err.Error())
					gerr.SetStatus(http.StatusInternalServerError)
					return nil, gerr
				}
			} else {
				found = true
				c.Versions[cbVersion] = cbv
			}
		}
	} else {
		cbv, found = c.Versions[cbVersion]
		if cbv != nil {
			datastore.ChkNilArray(cbv)
			if cbv.Recipes == nil {
				cbv.Recipes = make([]map[string]interface{}, 0)
			}
		}
	}

	if !found {
		err := util.Errorf("Cannot find a cookbook named %s with version %s", c.Name, cbVersion)
		err.SetStatus(http.StatusNotFound)
		return nil, err
	}
	return cbv, nil
}

func extractVerNums(cbVersion string) (maj, min, patch int64, err util.Gerror) {
	if _, err = util.ValidateAsVersion(cbVersion); err != nil {
		return 0, 0, 0, err
	}
	nums := strings.Split(cbVersion, ".")
	if len(nums) < 2 && len(nums) > 3 {
		err = util.Errorf("incorrect number of numbers in version string '%s': %d", cbVersion, len(nums))
		return 0, 0, 0, err
	}
	var vt int64
	var nerr error
	vt, nerr = strconv.ParseInt(nums[0], 0, 64)
	if nerr != nil {
		err = util.Errorf(nerr.Error())
		return 0, 0, 0, err
	}
	maj = vt
	vt, nerr = strconv.ParseInt(nums[1], 0, 64)
	if nerr != nil {
		err = util.Errorf(nerr.Error())
		return 0, 0, 0, err
	}
	min = vt
	if len(nums) == 3 {
		vt, nerr = strconv.ParseInt(nums[2], 0, 64)
		if nerr != nil {
			err = util.Errorf(nerr.Error())
			return 0, 0, 0, err
		}
		patch = vt
	} else {
		patch = 0
	}
	return maj, min, patch, nil
}

func (c *Cookbook) deleteHashes(fhashes []string) {
	/* And remove the unused hashes. Currently, sigh, this involves checking
	 * every cookbook. Probably will be easier with an actual database, I
	 * imagine. */
	ac := AllCookbooks()
	for _, cb := range ac {
		/* just move on if we don't find it somehow */
		// if we get to this cookbook, check the versions currently in
		// memory
		if cb.Name == c.Name {
			cb = c
		}
		for _, ver := range cb.sortedVersions() {
			verHash := ver.fileHashes()
			for _, vh := range verHash {
				for i, fh := range fhashes {
					/* If a hash in a deleted cookbook is
					 * in another cookbook, remove it from
					 * the hash to delete. Then we can break
					 * out. If we find that the hash we're
					 * comparing with is greater than this
					 * one in fhashes, also break out.
					 */
					if fh == vh {
						fhashes = delSliceElement(i, fhashes)
						break
					} else if fh > vh {
						break
					}
				}
			}
		}
	}
	/* And delete whatever file hashes we still have */
	filestore.DeleteHashes(fhashes)
}

// DeleteVersion deletes a particular version of a cookbook.
func (c *Cookbook) DeleteVersion(cbVersion string) util.Gerror {
	/* Check for existence */
	cbv, _ := c.GetVersion(cbVersion)
	if cbv == nil {
		err := util.Errorf("Version %s of cookbook %s does not exist to be deleted.", cbVersion, c.Name)
		err.SetStatus(http.StatusNotFound)
		return err
	}

	fhashes := cbv.fileHashes()

	if config.UsingDB() {
		err := cbv.deleteCookbookVersionSQL()
		if err != nil {
			return nil
		}
	}
	c.numVersions = nil

	delete(c.Versions, cbVersion)
	c.Save()
	c.deleteHashes(fhashes)

	return nil
}

// UpdateVersion updates a specific version of a cookbook.
func (cbv *CookbookVersion) UpdateVersion(cbvData map[string]interface{}, force string) util.Gerror {
	/* Allow force to update a frozen cookbook */
	if cbv.IsFrozen == true && force != "true" {
		err := util.Errorf("The cookbook %s at version %s is frozen. Use the 'force' option to override.", cbv.CookbookName, cbv.Version)
		err.SetStatus(http.StatusConflict)
		return err
	}

	fhashes := cbv.fileHashes()

	_, nerr := util.ValidateAsString(cbvData["cookbook_name"])
	if nerr != nil {
		if nerr.Error() == "Field 'name' missing" {
			nerr = util.Errorf("Field 'cookbook_name' missing")
		} else {
			nerr = util.Errorf("Field 'cookbook_name' invalid")
		}
		return nerr
	}

	/* Validation, validation, all is validation. */
	validElements := []string{"cookbook_name", "name", "version", "json_class", "chef_type", "definitions", "libraries", "attributes", "recipes", "providers", "resources", "templates", "root_files", "files", "frozen?", "metadata", "force"}
ValidElem:
	for k := range cbvData {
		for _, i := range validElements {
			if k == i {
				continue ValidElem
			}
		}
		err := util.Errorf("Invalid key %s in request body", k)
		return err
	}

	var verr util.Gerror
	cbvData["chef_type"], verr = util.ValidateAsFieldString(cbvData["chef_type"])
	if verr != nil {
		if verr.Error() == "Field 'name' nil" {
			cbvData["chef_type"] = cbv.ChefType
		} else {
			verr = util.Errorf("Field 'chef_type' invalid")
			return verr
		}
	} else {
		// Wait, what was I doing here?
		// if !util.ValidateEnvName(cbvData["chef_type"].(string)) {
		if cbvData["chef_type"].(string) != "cookbook_version" {
			verr = util.Errorf("Field 'chef_type' invalid")
			return verr
		}
	}

	cbvData["json_class"], verr = util.ValidateAsFieldString(cbvData["json_class"])
	if verr != nil {
		if verr.Error() == "Field 'name' nil" {
			cbvData["json_class"] = cbv.JSONClass
		} else {
			verr = util.Errorf("Field 'json_class' invalid")
			return verr
		}
	} else {
		if cbvData["json_class"].(string) != "Chef::CookbookVersion" {
			verr = util.Errorf("Field 'json_class' invalid")
			return verr
		}
	}

	cbvData["version"], verr = util.ValidateAsVersion(cbvData["version"])
	if verr != nil {
		verr = util.Errorf("Field 'version' invalid")
		return verr
	}
	if cbvData["version"].(string) == "0.0.0" && cbv.Version != "" {
		cbvData["version"] = cbv.Version
	}

	divs := []string{"definitions", "libraries", "attributes", "recipes", "providers", "resources", "templates", "root_files", "files"}
	for _, d := range divs {
		cbvData[d], verr = util.ValidateCookbookDivision(d, cbvData[d])
		if verr != nil {
			return verr
		}
	}
	cbvData["metadata"], verr = util.ValidateCookbookMetadata(cbvData["metadata"])
	if verr != nil {
		return verr
	}

	cbvData["frozen?"], verr = util.ValidateAsBool(cbvData["frozen?"])
	if verr != nil {
		return verr
	}

	/* Basic sanity checking */
	if cbvData["cookbook_name"].(string) != cbv.CookbookName {
		err := util.Errorf("Field 'cookbook_name' invalid")
		return err
	}
	if cbvData["name"].(string) != cbv.Name {
		err := util.Errorf("Field 'name' invalid")
		return err
	}
	if cbvData["version"].(string) != cbv.Version && cbvData["version"] != "0.0.0" {
		err := util.Errorf("Field 'version' invalid")
		return err
	}

	/* Update the data */
	/* With these next two, should we test for existence before setting? */
	cbv.ChefType = cbvData["chef_type"].(string)
	cbv.JSONClass = cbvData["json_class"].(string)
	cbv.Definitions = convertToCookbookDiv(cbvData["definitions"])
	cbv.Libraries = convertToCookbookDiv(cbvData["libraries"])
	cbv.Attributes = convertToCookbookDiv(cbvData["attributes"])
	cbv.Recipes = cbvData["recipes"].([]map[string]interface{})
	cbv.Providers = convertToCookbookDiv(cbvData["providers"])
	cbv.Resources = convertToCookbookDiv(cbvData["resources"])
	cbv.Templates = convertToCookbookDiv(cbvData["templates"])
	cbv.RootFiles = convertToCookbookDiv(cbvData["root_files"])
	cbv.Files = convertToCookbookDiv(cbvData["files"])
	if cbv.IsFrozen != true {
		cbv.IsFrozen = cbvData["frozen?"].(bool)
	}
	cbv.Metadata = cbvData["metadata"].(map[string]interface{})

	/* If we're using SQL, update this version in the DB. */
	if config.UsingDB() {
		if err := cbv.updateCookbookVersionSQL(); err != nil {
			return err
		}
	}

	/* Clean cookbook hashes */
	if len(fhashes) > 0 {
		// Get our parent. Bravely assuming that if it exists we exist.
		cbook, _ := Get(cbv.CookbookName)
		cbook.Versions[cbv.Version] = cbv
		cbook.deleteHashes(fhashes)
	}

	return nil
}

func convertToCookbookDiv(div interface{}) []map[string]interface{} {
	switch div := div.(type) {
	case []map[string]interface{}:
		return div
	default:
		return nil
	}
}

// Get the hashes of all files associated with a cookbook. Useful for comparing
// the files in a deleted cookbook version with the files in other versions to
// figure out which to remove and which to keep.
func (cbv *CookbookVersion) fileHashes() []string {
	/* Hmm. Weird as it seems, we seem to want length to be zero here so
	 * we can happily append. Otherwise we'll end up with a nil element. */
	fhashes := make([]string, 0)
	fhashes = append(fhashes, getAttrHashes(cbv.Definitions)...)
	fhashes = append(fhashes, getAttrHashes(cbv.Libraries)...)
	fhashes = append(fhashes, getAttrHashes(cbv.Attributes)...)
	fhashes = append(fhashes, getAttrHashes(cbv.Recipes)...)
	fhashes = append(fhashes, getAttrHashes(cbv.Providers)...)
	fhashes = append(fhashes, getAttrHashes(cbv.Resources)...)
	fhashes = append(fhashes, getAttrHashes(cbv.Templates)...)
	fhashes = append(fhashes, getAttrHashes(cbv.RootFiles)...)
	fhashes = append(fhashes, getAttrHashes(cbv.Files)...)

	/* Sort, then remove any duplicates */
	sort.Strings(fhashes)
	fhashes = removeDupHashes(fhashes)

	return fhashes
}

// ToJSON is a helper function that coverts the internal representation of a
// cookbook version to JSON in a way that knife and chef-client expect.
func (cbv *CookbookVersion) ToJSON(method string) map[string]interface{} {
	toJSON := make(map[string]interface{})
	toJSON["name"] = cbv.Name
	toJSON["cookbook_name"] = cbv.CookbookName
	if cbv.Version != "0.0.0" {
		toJSON["version"] = cbv.Version
	}
	toJSON["chef_type"] = cbv.ChefType
	toJSON["json_class"] = cbv.JSONClass
	toJSON["frozen?"] = cbv.IsFrozen
	// hmm.
	if cbv.Recipes != nil {
		toJSON["recipes"] = cbv.Recipes
	} else {
		toJSON["recipes"] = make([]map[string]interface{}, 0)
	}
	toJSON["metadata"] = cbv.Metadata

	/* Only send the other fields if something exists in them */
	/* Seriously, though, why *not* send the URL for the resources back
	 * with PUT, but *DO* send it with everything else? */
	if cbv.Providers != nil && len(cbv.Providers) != 0 {
		toJSON["providers"] = methodize(method, cbv.Providers)
	}
	if cbv.Definitions != nil && len(cbv.Definitions) != 0 {
		toJSON["definitions"] = methodize(method, cbv.Definitions)
	}
	if cbv.Libraries != nil && len(cbv.Libraries) != 0 {
		toJSON["libraries"] = methodize(method, cbv.Libraries)
	}
	if cbv.Attributes != nil && len(cbv.Attributes) != 0 {
		toJSON["attributes"] = methodize(method, cbv.Attributes)
	}
	if cbv.Resources != nil && len(cbv.Resources) != 0 {
		toJSON["resources"] = methodize(method, cbv.Resources)
	}
	if cbv.Templates != nil && len(cbv.Templates) != 0 {
		toJSON["templates"] = methodize(method, cbv.Templates)
	}
	if cbv.RootFiles != nil && len(cbv.RootFiles) != 0 {
		toJSON["root_files"] = methodize(method, cbv.RootFiles)
	}
	if cbv.Files != nil && len(cbv.Files) != 0 {
		toJSON["files"] = methodize(method, cbv.Files)
	}

	return toJSON
}

func methodize(method string, cbThing []map[string]interface{}) []map[string]interface{} {
	retHash := make([]map[string]interface{}, len(cbThing))
	baseURL := config.ServerBaseURL()
	r := regexp.MustCompile(`/file_store/`)
	for i, v := range cbThing {
		retHash[i] = make(map[string]interface{})
		chkSum := cbThing[i]["checksum"].(string)
		for k, j := range v {
			if method == "PUT" && k == "url" {
				continue
			}
			if k == "url" && r.MatchString(`/file_store/`) {
				retHash[i][k] = baseURL + "/file_store/" + chkSum
			} else {
				retHash[i][k] = j
			}
		}
	}
	return retHash
}

func getAttrHashes(attr []map[string]interface{}) []string {
	hashes := make([]string, len(attr))
	for i, v := range attr {
		/* Woo, type assertion again */
		switch h := v["checksum"].(type) {
		case string:
			hashes[i] = h
		case nil:
			/* anything special here? */

		default:
			// do we expect an err?
		}
	}
	return hashes
}

func removeDupHashes(fhashes []string) []string {
	for i, v := range fhashes {
		/* break if we're the last element */
		if i+1 >= len(fhashes) {
			break
		}
		/* If the current element is equal to the next one, remove
		 * the next element. */
		if v == fhashes[i+1] {
			fhashes = delSliceElement(i+1, fhashes)
		}
	}
	return fhashes
}

func delSliceElement(pos int, strs []string) []string {
	strs = append(strs[:pos], strs[pos+1:]...)
	return strs
}

// RecipeList provides a list of recipes in this cookbook version.
func (cbv *CookbookVersion) RecipeList() ([]string, util.Gerror) {
	recipeMeta := cbv.Recipes
	recipes := make([]string, len(recipeMeta))
	ci := 0
	/* Cobble the recipes together from the Recipes field */
	for _, r := range recipeMeta {
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
			rname = cbv.CookbookName
		} else {
			rname = fmt.Sprintf("%s::%s", cbv.CookbookName, rbase)
		}
		recipes[ci] = rname
		ci++
	}
	return recipes, nil
}

/* Version string functions to implement sorting */

func (v VersionStrings) Len() int {
	return len(v)
}

func (v VersionStrings) Swap(i, j int) {
	v[i], v[j] = v[j], v[i]
}

func (v VersionStrings) Less(i, j int) bool {
	return versionLess(v[i], v[j])
}

func versionLess(verA, verB string) bool {
	/* Chef cookbook versions are always to be in the form x.y.z (with x.y
	 * also allowed. This simplifies things a bit. */

	/* Easy comparison. False if they're equal. */
	if verA == verB {
		return false
	}

	/* Would caching the split strings ever be particularly worth it? */
	iVer := strings.Split(verA, ".")
	jVer := strings.Split(verB, ".")

	for q := 0; q < 3; q++ {
		/* If one of them doesn't actually exist, then obviously the
		 * other is bigger, and we're done. Of course this should only
		 * happen with the 3rd element. */
		if len(iVer) < q+1 {
			return true
		} else if len(jVer) < q+1 {
			return false
		}

		ic := iVer[q]
		jc := jVer[q]

		/* Otherwise, see if they're equal. If they're not, return the
		 * result of x < y. */
		ici, _ := strconv.Atoi(ic)
		jci, _ := strconv.Atoi(jc)
		if ici != jci {
			return ici < jci
		}
	}
	return false
}

/* Compares a version number against a constraint, like version 1.2.3 vs.
 * ">= 1.0.1". In this case, 1.2.3 passes. It would not satisfy "= 1.2.0" or
 * "< 1.0", though. */

func verConstraintCheck(verA, verB, op string) string {
	switch op {
	case "=":
		if verA == verB {
			return "ok"
		} else if versionLess(verA, verB) {
			/* If we want equality and verA is less than
			 * version b, since the version list is sorted
			 * in descending order we've missed our chance.
			 * So, break out. */
			return "break"
		} else {
			return "skip"
		}
	case ">":
		if verA == verB || versionLess(verA, verB) {
			return "break"
		}
		return "ok"
	case "<":
		/* return skip here because we might find what we want
		 * later. */
		if verA == verB || !versionLess(verA, verB) {
			return "skip"
		}
		return "ok"
	case ">=":
		if !versionLess(verA, verB) {
			return "ok"
		}
		return "break"
	case "<=":
		if verA == verB || versionLess(verA, verB) {
			return "ok"
		}
		return "skip"
	case "~>":
		/* only check pessimistic constraints if they can
		 * possibly be valid. */
		if versionLess(verA, verB) {
			return "break"
		}
		var upperBound string
		pv := strings.Split(verB, ".")
		if len(pv) == 3 {
			uver, _ := strconv.Atoi(pv[1])
			uver++
			upperBound = fmt.Sprintf("%s.%d", pv[0], uver)
		} else {
			uver, _ := strconv.Atoi(pv[0])
			uver++
			upperBound = fmt.Sprintf("%d.0", uver)
		}
		if !versionLess(verA, verB) && versionLess(verA, upperBound) {

			return "ok"
		}
		return "skip"
	default:
		return "invalid"
	}
}
