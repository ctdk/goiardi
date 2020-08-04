/* Cookbooks! The ultimate building block of any chef run. */

/*
 * Copyright (c) 2013-2017, Jeremy Bingham (<jeremy@goiardi.gl>)
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
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/datastore"
	"github.com/ctdk/goiardi/depgraph"
	"github.com/ctdk/goiardi/filestore"
	"github.com/ctdk/goiardi/types"
	"github.com/ctdk/goiardi/util"
	gversion "github.com/hashicorp/go-version"
	"github.com/tideland/golib/logger"
)

// cookbook divisions, when resolving cookbook dependencies, that must be filled
// with a zero length array (not nil) when they are returned.
var chkDiv = [...]string{"definitions", "libraries", "attributes", "providers", "resources", "templates", "root_files", "files"}

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
	CookbookName string         `json:"cookbook_name"`
	Name         string         `json:"name"`
	Version      string         `json:"version"`
	ChefType     string         `json:"chef_type"`
	JSONClass    string         `json:"json_class"`
	Definitions  []types.Files  `json:"definitions"`
	Libraries    []types.Files  `json:"libraries"`
	Attributes   []types.Files  `json:"attributes"`
	Recipes      []types.Files  `json:"recipes"`
	Providers    []types.Files  `json:"providers"`
	Resources    []types.Files  `json:"resources"`
	Templates    []types.Files  `json:"templates"`
	RootFiles    []types.Files  `json:"root_files"`
	Files        []types.Files  `json:"files"`
	IsFrozen     bool           `json:"frozen?"`
	Metadata     types.Metadata `json:"metadata"`
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
	if !util.ValidateName(name) {
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
func AllCookbooks() (cookbooks []*Cookbook, err error) {
	if config.UsingDB() {
		cookbooks = allCookbooksSQL()
		err = massPopulateVersionsSQL(cookbooks)
		if err != nil {
			return nil, err
		}
		return cookbooks, nil
	}
	cookbookList := GetList()
	for _, c := range cookbookList {
		cb, found, err := Get(c)
		switch {
		case !found:
			logger.Debugf("Curious. Cookbook %s was in the cookbook list, but wasn't found when fetched. Continuing.", c)
			continue
		case err != nil:
			// todo: this needs to be handled in a much better way.
			// AllCookbooks() should return an err which should be
			// checked above. Leaving this for another day
			logger.Errorf(err.Error())
			continue
		}
		cookbooks = append(cookbooks, cb)
	}
	return cookbooks, nil
}

// Get a cookbook.
func Get(name string) (cookbook *Cookbook, found bool, gerror util.Gerror) {
	//if enabled, fetch cookbooks from database
	if config.UsingDB() {
		cookbook, err := getCookbookSQL(name)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, false, nil
			}
			gerr := util.CastErr(err)
			gerr.SetStatus(http.StatusInternalServerError)
			return nil, false, gerr
		}
		return cookbook, true, nil
	}

	//get the cookbook from internal datastore
	ds := datastore.New()
	var c interface{}
	c, found = ds.Get("cookbook", name)
	if !found {
		return nil, false, nil
	}
	// this should never happen, but still lets put a check in place
	if c == nil {
		err := util.Errorf("a cookbook %s has been reported as found but it is null", name)
		err.SetStatus(http.StatusNotFound)
		return nil, false, err
	}

	cookbook = c.(*Cookbook)
	/* hrm. */
	if config.Config.UseUnsafeMemStore {
		for _, v := range cookbook.Versions {
			datastore.ChkNilArray(v)
		}
	}
	return cookbook, true, nil
}

// DoesExist checks if the cookbook in question exists or not
func DoesExist(cookbookName string) (bool, util.Gerror) {
	var found bool
	if config.UsingDB() {
		var cerr error
		found, cerr = checkForCookbookSQL(datastore.Dbh, cookbookName)
		if cerr != nil {
			err := util.Errorf(cerr.Error())
			err.SetStatus(http.StatusInternalServerError)
			return false, err
		}
	} else {
		ds := datastore.New()
		_, found = ds.Get("cookbook", cookbookName)
	}
	return found, nil
}

// Save a cookbook to the in-memory data store or database.
func (c *Cookbook) Save() error {
	if config.Config.UseMySQL {
		return c.saveCookbookMySQL()
	} else if config.Config.UsePostgreSQL {
		return c.saveCookbookPostgreSQL()
	} else {
		ds := datastore.New()
		ds.Set("cookbook", c.Name, c)
	}
	return nil
}

// Delete a coookbook.
func (c *Cookbook) Delete() error {
	if config.UsingDB() {
		return c.deleteCookbookSQL()
	}
	ds := datastore.New()
	ds.Delete("cookbook", c.Name)

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
func CookbookLister(numResults interface{}) (map[string]interface{}, error) {
	if config.UsingDB() {
		return cookbookListerSQL(numResults), nil //todo: proper error reporting.
	}
	cr := make(map[string]interface{})
	cookbooks, err := AllCookbooks()
	if err != nil {
		return nil, err
	}
	for _, cb := range cookbooks {
		cr[cb.Name] = cb.InfoHash(numResults)
	}
	return cr, nil
}

// CookbookLatest returns the URL of the latest version of each cookbook on the
// server.
func CookbookLatest() (map[string]interface{}, error) {
	latest := make(map[string]interface{})
	if config.UsingDB() {
		cs, err := CookbookLister("")
		if err != nil {
			return nil, err
		}
		for name, cbdata := range cs {
			if len(cbdata.(map[string]interface{})["versions"].([]interface{})) > 0 {
				latest[name] = cbdata.(map[string]interface{})["versions"].([]interface{})[0].(map[string]string)["url"]
			}
		}
		return latest, nil
	}
	cbs, err := AllCookbooks()
	if err != nil {
		return nil, err
	}
	for _, cb := range cbs {
		latest[cb.Name] = util.CustomObjURL(cb, cb.LatestVersion().Version)
	}
	return latest, nil
}

// CookbookRecipes returns a list of all the recipes on the server in the latest
// version of each cookbook.
func CookbookRecipes() ([]string, util.Gerror) {
	if config.UsingDB() {
		return cookbookRecipesSQL()
	}
	rlist := make([]string, 0)
	cookbooks, err := AllCookbooks()
	if err != nil {
		gerr := util.Errorf("cannot get all cookbooks")
		gerr.SetStatus(http.StatusInternalServerError)
		return nil, gerr
	}
	for _, cb := range cookbooks {
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
	nodes := make(map[string]*depgraph.Noun)
	runListRef := make([]string, len(runList))

	for i, cbV := range runList {
		var cbName string
		var constraint string
		cx := strings.Split(cbV, "@")
		cbName = strings.Split(cx[0], "::")[0]
		if len(cx) == 2 {
			constraint = fmt.Sprintf("= %s", cx[1])
		}
		nodes[cbName] = &depgraph.Noun{Name: cbName}
		meta := &depMeta{}
		if constraint != "" {
			q, _ := gversion.NewConstraint(constraint)
			meta.constraint = versionConstraint(q)
		}
		nodes[cbName].Meta = meta
		runListRef[i] = cbName
	}

	for k, ec := range envConstraints {
		if _, found := nodes[k]; !found {
			continue
		}
		appendConstraint(&nodes[k].Meta.(*depMeta).constraint, ec)
	}

	graphRoot := &depgraph.Noun{Name: "^runlist_root^"}
	g := &depgraph.Graph{Name: "runlist", Root: graphRoot}

	// fill in constraints for runlist deps now
	for k, n := range nodes {
		d := &depgraph.Dependency{Name: fmt.Sprintf("%s-%s", g.Name, k), Source: graphRoot, Target: n, Constraints: []depgraph.Constraint{versionConstraint(n.Meta.(*depMeta).constraint)}}
		graphRoot.Deps = append(graphRoot.Deps, d)
	}

	cbShelf := make(map[string]*Cookbook)
	for _, cbName := range runListRef {
		if _, found := cbShelf[cbName]; found || nodes[cbName].Meta.(*depMeta).notFound {
			continue
		}
		cb, found, err := Get(cbName)
		switch {
		case !found:
			nodes[cbName].Meta.(*depMeta).notFound = true
			continue
		case err != nil:
			//todo: not ideal return since the return code would be precondition failed, but still better than nothing for now
			return nil, err
		}
		cbShelf[cbName] = cb
		cbv := cb.latestMultiConstraint(nodes[cbName].Meta.(*depMeta).constraint)
		if cbv == nil {
			nodes[cbName].Meta.(*depMeta).noVersion = true
			continue
		}
		nodes[cbName].Meta.(*depMeta).version = cbv.Version
		cbv.getDependencies(g, nodes, cbShelf)
	}
	nouns := make([]*depgraph.Noun, 1)
	nouns[0] = graphRoot
	g.Nouns = nouns

	cerr := g.CheckConstraints()

	if cerr != nil {
		err := &DependsError{cerr.(*depgraph.ConstraintError)}
		return nil, err
	}

	cookbookDeps := make(map[string]interface{}, len(cbShelf))
	for k, c := range cbShelf {
		constraints := nodes[k].Meta.(*depMeta).constraint
		cbv := c.latestMultiConstraint(constraints)
		if cbv == nil {
			err := fmt.Errorf("Somehow, and this shouldn't have beenable to happen at this stage, no versions of %s satisfied the constraints '%s'!", c.Name, constraints.String())
			return nil, err
		}
		gcbvJSON := cbv.ToJSON("POST")

		for _, cd := range chkDiv {
			if gcbvJSON[cd] == nil {
				gcbvJSON[cd] = make([]map[string]interface{}, 0)
			}
		}
		cookbookDeps[cbv.CookbookName] = gcbvJSON
	}
	return cookbookDeps, nil
}

func (c *Cookbook) latestMultiConstraint(constraints versionConstraint) *CookbookVersion {
	var cbv *CookbookVersion
	if constraints == nil {
		cbv = c.LatestVersion()
	} else {
		cbversions := c.sortedVersions()
	Ver:
		for _, cver := range cbversions {
			v, _ := gversion.NewVersion(cver.Version)
			for _, cs := range constraints {
				if !cs.Check(v) {
					continue Ver
				}
				cbv = cver
				break Ver
			}
		}
	}
	return cbv
}

func (c *Cookbook) badConstraints(constraints versionConstraint) []string {
	bad := make([]string, 0, len(constraints))
	if constraints == nil {
		return bad
	}
	cbversions := c.sortedVersions()
	for _, cs := range constraints {
		for _, cver := range cbversions {
			v, _ := gversion.NewVersion(cver.Version)
			if !cs.Check(v) {
				bad = append(bad, cs.String())
				break
			}
		}
	}
	return bad
}

func (cbv *CookbookVersion) getDependencies(g *depgraph.Graph, nodes map[string]*depgraph.Noun, cbShelf map[string]*Cookbook) {
	for r, c2 := range cbv.Metadata.Dependencies {
		if _, ok := nodes[r]; ok {
			if nodes[r].Meta.(*depMeta).noVersion || nodes[r].Meta.(*depMeta).notFound {
				continue
			}
		}
		var depCb *Cookbook
		var err util.Gerror
		var found bool

		if _, ok := nodes[r]; !ok {
			nodes[r] = &depgraph.Noun{Name: r, Meta: &depMeta{}}
		}
		dep, depPos, dt := checkDependency(nodes[cbv.CookbookName], r)
		if dep == nil {
			dep = &depgraph.Dependency{Name: fmt.Sprintf("%s-%s", cbv.CookbookName, r), Source: nodes[cbv.CookbookName], Target: nodes[r]}
		}
		depCons, _ := gversion.NewConstraint(c2)
		dep.Constraints = []depgraph.Constraint{versionConstraint(depCons)}
		if !dt || nodes[cbv.CookbookName].Deps == nil {
			nodes[cbv.CookbookName].Deps = append(nodes[cbv.CookbookName].Deps, dep)
		} else {
			nodes[cbv.CookbookName].Deps[depPos] = dep
		}

		if depCb, found = cbShelf[r]; !found {
			depCb, found, err = Get(r)
			switch {
			case !found:
				nodes[r].Meta.(*depMeta).notFound = true
				appendConstraint(&nodes[r].Meta.(*depMeta).constraint, c2)
				continue
			case err != nil:
				//todo: we should really return an error from here, for now I am going to act as if cookbook was not
				//found and report an error additionally
				logger.Errorf("Cannot get a cookbook %s", err)
				nodes[r].Meta.(*depMeta).notFound = true
				appendConstraint(&nodes[r].Meta.(*depMeta).constraint, c2)
				continue
			}
		} else {
			// see if this constraint and a dependency for this
			// cookbook is already in place. If it is, go ahead and
			// move along, we've already been here.
			if dt && constraintPresent(nodes[r].Meta.(*depMeta).constraint, c2) {
				continue
			}
		}
		appendConstraint(&nodes[r].Meta.(*depMeta).constraint, c2)

		cbShelf[r] = depCb
		depCbv := depCb.latestMultiConstraint(nodes[r].Meta.(*depMeta).constraint)
		if depCbv == nil {
			nodes[r].Meta.(*depMeta).noVersion = true
			continue
		}
		if nodes[r].Meta.(*depMeta).version != "" && nodes[r].Meta.(*depMeta).version != depCbv.Version {
			// Remove any dependencies for this cookbook's node.
			// They'll be filled in
			nodes[r].Deps = make([]*depgraph.Dependency, 0)
		}

		nodes[r].Meta.(*depMeta).version = depCbv.Version

		depCbv.getDependencies(g, nodes, cbShelf)
	}
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
func Universe() (map[string]map[string]interface{}, error) {
	if config.UsingDB() {
		return universeSQL(), nil
	}
	universe := make(map[string]map[string]interface{})

	cookbooks, err := AllCookbooks()
	if err != nil {
		return nil, err
	}
	for _, cb := range cookbooks {
		universe[cb.Name] = cb.universeFormat()
	}
	return universe, nil
}

// universeFormat returns a sorted list of this cookbook's versions, formatted
// to be compatible with the supermarket/berks /universe endpoint.
func (c *Cookbook) universeFormat() map[string]interface{} {
	u := make(map[string]interface{})
	var dependencies []string
	for _, cbv := range c.sortedVersions() {
		v := make(map[string]interface{})
		v["location_path"] = util.CustomObjURL(c, cbv.Version)
		v["location_type"] = "chef_server"
		dependencies = []string{}
		for _, value := range cbv.Metadata.Dependencies {
			dependencies = append(dependencies, value)
		}
		v["dependencies"] = strings.Join(dependencies, ",") //todo: check if a proper value is selected
		u[cbv.Version] = v
	}
	return u
}

/* CookbookVersion methods and functions */

func (c *Cookbook) NewVersion(cbVersion string, newCbVersion CookbookVersion) (*CookbookVersion, util.Gerror) {
	cbv := &CookbookVersion{
		CookbookName: c.Name,
		Version:      cbVersion,
		Name:         fmt.Sprintf("%s-%s", c.Name, cbVersion),
		ChefType:     "cookbook_version",
		JSONClass:    "Chef::CookbookVersion",
		IsFrozen:     false,
		cookbookID:   c.id, // should be ok even with in-mem
	}
	err := cbv.UpdateVersion(newCbVersion, false)
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

// NewVersionFromGenericData creates a new version of the cookbook.
// todo: This was left here because of refactoring (from anon json to a proper datastructs). Currently the only
// place this is used for is import/export. We need to refactor those 2 as well and delete the funct below.
func (c *Cookbook) NewVersionFromGenericData(cbVersion string, cbvData map[string]interface{}) (*CookbookVersion, util.Gerror) {
	return nil, nil
	//cbv, err := c.GetVersion(cbVersion)
	//if err != nil {
	//	return nil, err
	//}
	//if cbv != nil {
	//	err := util.Errorf("Version %s of cookbook %s already exists, and shouldn't be created like this. Use UpdateVersion instead.", cbVersion, c.Name)
	//	err.SetStatus(http.StatusConflict)
	//	return nil, err
	//}
	//
	//cbv = &CookbookVersion{
	//	CookbookName: c.Name,
	//	Version:      cbVersion,
	//	Name:         fmt.Sprintf("%s-%s", c.Name, cbVersion),
	//	ChefType:     "cookbook_version",
	//	JSONClass:    "Chef::CookbookVersion",
	//	IsFrozen:     false,
	//	cookbookID:   c.id, // should be ok even with in-mem
	//}
	//err = cbv.UpdateVersion(cbvData, "")
	//if err != nil {
	//	return nil, err
	//}
	///* And, dur, add it to the versions */
	//c.Versions[cbVersion] = cbv
	//
	//c.numVersions = nil
	//c.UpdateLatestVersion()
	//c.Save()
	//return cbv, nil
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
			cbvTmp, err := c.getCookbookVersionSQL(cbVersion)
			if err != nil {
				//no queries, return without error
				if err == sql.ErrNoRows {
					return nil, nil
				}
				// something happened, return a proper error
				gerr := util.Errorf(err.Error())
				gerr.SetStatus(http.StatusInternalServerError)
				return nil, gerr
			}
			//all good, we found the current version
			c.Versions[cbVersion] = &cbvTmp
		}
	}
	//get what we have in memory by this point (hopefully datastore has them in place by this point)
	if cbv, found = c.Versions[cbVersion]; found {
		return cbv, nil
	}
	return nil, nil
}

// DoesVersionExist checks if a particular version of a cookbook exists
func (c *Cookbook) DoesVersionExist(cbVersion string) (bool, util.Gerror) {
	cbv, err := c.GetVersion(cbVersion)
	if err != nil {
		cerr := util.CastErr(err)
		cerr.SetStatus(http.StatusInternalServerError)
		return false, err
	}
	return cbv != nil, nil
}

func (c *Cookbook) deleteHashes(hashesToDelete map[string]bool) error {
	/* And remove the unused hashes. Currently, sigh, this involves checking
	 * every cookbook. Probably will be easier with an actual database, I
	 * imagine. */
	allCookbooks, err := AllCookbooks()
	if err != nil {
		return err
	}
	for _, cb := range allCookbooks {
		// for every version in a given cookbook
		for _, cookbookVersion := range cb.Versions {
			for key, _ := range cookbookVersion.fileHashesMap() {
				/* If a hash in a deleted cookbook is
				 * in another cookbook, remove it from
				 * the hash to delete.*/
				if _, ok := hashesToDelete[key]; ok {
					delete(hashesToDelete, key)
				}
			}
		}
	}
	if len(hashesToDelete) == 0 {
		return nil
	}

	/* And delete whatever file hashes we still have */
	if config.Config.UseS3Upload {
		util.S3DeleteHashes(hashesToDelete)
	} else {
		filestore.DeleteHashes(hashesToDelete)
	}
	return nil
}

// DeleteVersion deletes a particular version of a cookbook.
func (c *Cookbook) DeleteVersion(cbVersion string) util.Gerror {
	/* Check for existence */
	cbv, err := c.GetVersion(cbVersion)
	if err != nil {
		err := util.Errorf(err.Error())
		err.SetStatus(http.StatusInternalServerError)
		return err
	}
	if cbv == nil {
		err := util.Errorf("Version %s of cookbook %s does not exist to be deleted.", cbVersion, c.Name)
		err.SetStatus(http.StatusNotFound)
		return err
	}

	fhashes := cbv.fileHashesMap()

	if config.UsingDB() {
		dbErr := cbv.deleteCookbookVersionSQL()
		if dbErr != nil {
			err := util.Errorf(dbErr.Error())
			err.SetStatus(http.StatusInternalServerError)
			return err
		}
	}
	c.numVersions = nil

	delete(c.Versions, cbVersion)
	c.Save()
	c.deleteHashes(fhashes)

	return nil
}

// UpdateVersion updates a specific version of a cookbook.
func (cbv *CookbookVersion) UpdateVersion(newCbVersion CookbookVersion, force bool) util.Gerror {
	/* Allow force to update a frozen cookbook */
	if cbv.IsFrozen == true && !force {
		err := util.Errorf("The cookbook %s at version %s is frozen. Use the 'force' option to override.", cbv.CookbookName, cbv.Version)
		err.SetStatus(http.StatusConflict)
		return err
	}

	//get preexisting file hashes
	fhashes := cbv.fileHashesMap()

	// validate cookbook name
	if newCbVersion.Name == "" {
		err := util.Errorf("Field 'cookbook_name' missing")
		err.SetStatus(http.StatusBadRequest)
		return err
	}

	// validate cheftype
	switch newCbVersion.ChefType {
	case "":
		newCbVersion.ChefType = cbv.ChefType
	case "cookbook_version":
		break
	default:
		verr := util.Errorf("Field 'chef_type' invalid")
		verr.SetStatus(http.StatusBadRequest)
		return verr
	}

	// validate jsonClass
	switch newCbVersion.JSONClass {
	case "":
		newCbVersion.JSONClass = cbv.JSONClass
	case "Chef::CookbookVersion":
		break
	default:
		verr := util.Errorf("Field 'json_class' invalid")
		verr.SetStatus(http.StatusBadRequest)
		return verr
	}

	if (newCbVersion.Version == "" || newCbVersion.Version == "0.0.0") && cbv.Version != "" {
		newCbVersion.Version = cbv.Version
	}

	//TODO: CHECK!!!
	//divs := []string{"definitions", "libraries", "attributes", "recipes", "providers", "resources", "templates", "root_files", "files"}
	//for _, d := range divs {
	//	cbvData[d], verr = util.ValidateCookbookDivision(d, cbvData[d])
	//	if verr != nil {
	//		return verr
	//	}
	//}

	verr := util.ValidateCookbookMetadata(newCbVersion.Metadata)
	if verr != nil {
		verr.SetStatus(http.StatusBadRequest)
		return verr
	}

	/* Basic sanity checking */
	if newCbVersion.CookbookName != cbv.CookbookName {
		err := util.Errorf("Field 'cookbook_name' invalid")
		return err
	}
	if newCbVersion.Name != cbv.Name {
		err := util.Errorf("Field 'name' invalid")
		return err
	}
	if newCbVersion.Version != cbv.Version && newCbVersion.Version != "0.0.0" {
		err := util.Errorf("Field 'version' invalid")
		return err
	}

	/* Update the data */
	/* With these next two, should we test for existence before setting? */
	cbv.ChefType = newCbVersion.ChefType
	cbv.JSONClass = newCbVersion.JSONClass
	cbv.Definitions = newCbVersion.Definitions
	cbv.Libraries = newCbVersion.Libraries
	cbv.Attributes = newCbVersion.Attributes
	cbv.Recipes = newCbVersion.Recipes
	cbv.Providers = newCbVersion.Providers
	cbv.Resources = newCbVersion.Resources
	cbv.Templates = newCbVersion.Templates
	cbv.RootFiles = newCbVersion.RootFiles
	cbv.Files = newCbVersion.Files
	if cbv.IsFrozen != true {
		cbv.IsFrozen = newCbVersion.IsFrozen
	}
	cbv.Metadata = newCbVersion.Metadata

	/* If we're using SQL, update this version in the DB. */
	if config.UsingDB() {
		if err := cbv.updateCookbookVersionSQL(); err != nil {
			return err
		}
	}

	/* Clean cookbook hashes */
	// check if we have created some orphaned hashes with this cookbook update. If so, delete them.
	if len(fhashes) > 0 {
		cookbook, found, err := Get(cbv.CookbookName)
		switch {
		case !found:
			gerr := util.Errorf("cannot get a cookbook with name %s", cbv.CookbookName)
			gerr.SetStatus(http.StatusInternalServerError)
			return gerr
		case err != nil:
			return err
		}
		cookbook.Versions[cbv.Version] = cbv
		cerr := cookbook.deleteHashes(fhashes)
		if cerr != nil {
			gerr := util.Errorf("cannot delete hashes for a cookbook %s", cbv.CookbookName)
			gerr.SetStatus(http.StatusInternalServerError)
			return gerr
		}
	}

	return nil
}

func convertToCookbookDiv(div interface{}) []types.Files {
	switch div := div.(type) {
	case []types.Files:
		return div
	default:
		return []types.Files{}
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

// fileHashesMap gets the hashes of all files associated with a cookbook and return them as a map.
// Useful for comparing the files in a deleted cookbook version with the files in other versions to
// figure out which to remove and which to keep.
func (cbv *CookbookVersion) fileHashesMap() map[string]bool {
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

	//convert to hashmap (this will also take care of any duplicates)
	result := make(map[string]bool)
	for _, v := range fhashes {
		result[v] = false
	}
	return result
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
	toJSON["recipes"] = cbv.Recipes
	toJSON["metadata"] = cbv.Metadata

	/* Only send the other fields if something exists in them */
	/* Seriously, though, why *not* send the URL for the resources back
	 * with PUT, but *DO* send it with everything else? */
	if cbv.Providers != nil && len(cbv.Providers) != 0 {
		toJSON["providers"] = cbv.Providers
	}
	if cbv.Definitions != nil && len(cbv.Definitions) != 0 {
		toJSON["definitions"] = cbv.Definitions
	}
	if cbv.Libraries != nil && len(cbv.Libraries) != 0 {
		toJSON["libraries"] = cbv.Libraries
	}
	if cbv.Attributes != nil && len(cbv.Attributes) != 0 {
		toJSON["attributes"] = cbv.Attributes
	}
	if cbv.Resources != nil && len(cbv.Resources) != 0 {
		toJSON["resources"] = cbv.Resources
	}
	if cbv.Templates != nil && len(cbv.Templates) != 0 {
		toJSON["templates"] = cbv.Templates
	}
	if cbv.RootFiles != nil && len(cbv.RootFiles) != 0 {
		toJSON["root_files"] = cbv.RootFiles
	}
	if cbv.Files != nil && len(cbv.Files) != 0 {
		toJSON["files"] = cbv.Files
	}

	return toJSON
}

func getAttrHashes(attr []types.Files) []string {
	hashes := make([]string, 0, len(attr))
	for _, file := range attr {
		hashes = append(hashes, file.Checksum)
	}
	return hashes
}

func removeDupHashes(fhashes []string) []string {
	// needed this functionality elsewhere, so it's been moved to util.
	// Keeping this as a wrapper for simplicity.
	return util.RemoveDupStrings(fhashes)
}

// RecipeList provides a list of recipes in this cookbook version.
func (cbv *CookbookVersion) RecipeList() ([]string, util.Gerror) {
	recipeMeta := cbv.Recipes
	recipes := make([]string, len(recipeMeta))
	ci := 0
	/* Cobble the recipes together from the Recipes field */
	for _, r := range recipeMeta {
		rm := regexp.MustCompile(`(.*?)\.rb`)
		rfind := rm.FindStringSubmatch(r.Name)
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
