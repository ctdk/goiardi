/* Environments. */

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

// Package environment provides... environments. They're like roles, but more
// so, except without run lists. They're a convenient way to share many
// attributes and cookbook version constraints among many servers.
package environment

import (
	"database/sql"
	"fmt"
	"net/http"
	"sort"

	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/cookbook"
	"github.com/ctdk/goiardi/datastore"
	"github.com/ctdk/goiardi/indexer"
	"github.com/ctdk/goiardi/util"
)

// ChefEnvironment is a collection of attributes and cookbook versions for
// organizing how nodes are deployed.
type ChefEnvironment struct {
	Name             string                 `json:"name"`
	ChefType         string                 `json:"chef_type"`
	JSONClass        string                 `json:"json_class"`
	Description      string                 `json:"description"`
	Default          map[string]interface{} `json:"default_attributes"`
	Override         map[string]interface{} `json:"override_attributes"`
	CookbookVersions map[string]string      `json:"cookbook_versions"`
}

// New creates a new environment, returning an error if the environment already
// exists or you try to create an environment named "_default".
func New(name string) (*ChefEnvironment, util.Gerror) {
	if !util.ValidateEnvName(name) {
		err := util.Errorf("Field 'name' invalid")
		err.SetStatus(http.StatusBadRequest)
		return nil, err
	}

	var found bool
	if config.UsingDB() {
		var eerr error
		found, eerr = checkForEnvironmentSQL(datastore.Dbh, name)
		if eerr != nil {
			err := util.CastErr(eerr)
			err.SetStatus(http.StatusInternalServerError)
			return nil, err
		}
	} else {
		ds := datastore.New()
		_, found = ds.Get("env", name)
	}
	if found || name == "_default" {
		err := util.Errorf("Environment already exists")
		return nil, err
	}

	env := &ChefEnvironment{
		Name:             name,
		ChefType:         "environment",
		JSONClass:        "Chef::Environment",
		Default:          map[string]interface{}{},
		Override:         map[string]interface{}{},
		CookbookVersions: map[string]string{},
	}
	return env, nil
}

// NewFromJSON creates a new environment from JSON uploaded to the server.
func NewFromJSON(jsonEnv map[string]interface{}) (*ChefEnvironment, util.Gerror) {
	env, err := New(jsonEnv["name"].(string))
	if err != nil {
		return nil, err
	}
	err = env.UpdateFromJSON(jsonEnv)
	if err != nil {
		return nil, err
	}
	return env, nil
}

// UpdateFromJSON updates an existing environment from JSON uploaded to the
// server.
func (e *ChefEnvironment) UpdateFromJSON(jsonEnv map[string]interface{}) util.Gerror {
	if e.Name != jsonEnv["name"].(string) {
		err := util.Errorf("Environment name %s and %s from JSON do not match", e.Name, jsonEnv["name"].(string))
		return err
	} else if e.Name == "_default" {
		err := util.Errorf("The '_default' environment cannot be modified.")
		err.SetStatus(http.StatusMethodNotAllowed)
		return err
	}

	/* Validations */
	validElements := []string{"name", "chef_type", "json_class", "description", "default_attributes", "override_attributes", "cookbook_versions"}
ValidElem:
	for k := range jsonEnv {
		for _, i := range validElements {
			if k == i {
				continue ValidElem
			}
		}
		err := util.Errorf("Invalid key %s in request body", k)
		return err
	}

	var verr util.Gerror

	attrs := []string{"default_attributes", "override_attributes"}
	for _, a := range attrs {
		jsonEnv[a], verr = util.ValidateAttributes(a, jsonEnv[a])
		if verr != nil {
			return verr
		}
	}

	jsonEnv["json_class"], verr = util.ValidateAsFieldString(jsonEnv["json_class"])
	if verr != nil {
		if verr.Error() == "Field 'name' nil" {
			jsonEnv["json_class"] = e.JSONClass
		} else {
			return verr
		}
	} else {
		if jsonEnv["json_class"].(string) != "Chef::Environment" {
			verr = util.Errorf("Field 'json_class' invalid")
			return verr
		}
	}

	jsonEnv["chef_type"], verr = util.ValidateAsFieldString(jsonEnv["chef_type"])
	if verr != nil {
		if verr.Error() == "Field 'name' nil" {
			jsonEnv["chef_type"] = e.ChefType
		} else {
			return verr
		}
	} else {
		if jsonEnv["chef_type"].(string) != "environment" {
			verr = util.Errorf("Field 'chef_type' invalid")
			return verr
		}
	}

	jsonEnv["cookbook_versions"], verr = util.ValidateAttributes("cookbook_versions", jsonEnv["cookbook_versions"])
	if verr != nil {
		return verr
	}
	for k, v := range jsonEnv["cookbook_versions"].(map[string]interface{}) {
		if !util.ValidateEnvName(k) || k == "" {
			merr := util.Errorf("Cookbook name %s invalid", k)
			merr.SetStatus(http.StatusBadRequest)
			return merr
		}

		if v == nil {
			verr = util.Errorf("Invalid version number")
			return verr
		}
		_, verr = util.ValidateAsConstraint(v)
		if verr != nil {
			/* try validating as a version */
			v, verr = util.ValidateAsVersion(v)
			if verr != nil {
				return verr
			}
		}
	}

	jsonEnv["description"], verr = util.ValidateAsString(jsonEnv["description"])
	if verr != nil {
		if verr.Error() == "Field 'name' missing" {
			jsonEnv["description"] = ""
		} else {
			return verr
		}
	}

	e.ChefType = jsonEnv["chef_type"].(string)
	e.JSONClass = jsonEnv["json_class"].(string)
	e.Description = jsonEnv["description"].(string)
	e.Default = jsonEnv["default_attributes"].(map[string]interface{})
	e.Override = jsonEnv["override_attributes"].(map[string]interface{})
	/* clear out, then loop over the cookbook versions */
	e.CookbookVersions = make(map[string]string, len(jsonEnv["cookbook_versions"].(map[string]interface{})))
	for c, v := range jsonEnv["cookbook_versions"].(map[string]interface{}) {
		e.CookbookVersions[c] = v.(string)
	}

	return nil
}

// Get an environment.
func Get(envName string) (*ChefEnvironment, util.Gerror) {
	if envName == "_default" {
		return defaultEnvironment(), nil
	}
	var env *ChefEnvironment
	var found bool
	if config.UsingDB() {
		var err error
		env, err = getEnvironmentSQL(envName)
		if err != nil {
			var gerr util.Gerror
			if err != sql.ErrNoRows {
				gerr = util.CastErr(err)
				gerr.SetStatus(http.StatusInternalServerError)
				return nil, gerr
			}
			found = false
		} else {
			found = true
		}
	} else {
		ds := datastore.New()
		var e interface{}
		e, found = ds.Get("env", envName)
		if e != nil {
			env = e.(*ChefEnvironment)
		}
	}
	if !found {
		err := util.Errorf("Cannot load environment %s", envName)
		err.SetStatus(http.StatusNotFound)
		return nil, err
	}

	return env, nil
}

// DoesExist checks if the environment in question exists or not
func DoesExist(environmentName string) (bool, util.Gerror) {
	var found bool
	if config.UsingDB() {
		var cerr error
		found, cerr = checkForEnvironmentSQL(datastore.Dbh, environmentName)
		if cerr != nil {
			err := util.Errorf(cerr.Error())
			err.SetStatus(http.StatusInternalServerError)
			return false, err
		}
	} else {
		ds := datastore.New()
		_, found = ds.Get("env", environmentName)
	}
	return found, nil
}

// GetMulti gets multiple environmets from a given slice of environment names.
func GetMulti(envNames []string) ([]*ChefEnvironment, util.Gerror) {
	var envs []*ChefEnvironment
	if config.UsingDB() {
		var err error
		envs, err = getMultiSQL(envNames)
		if err != nil && err != sql.ErrNoRows {
			return nil, util.CastErr(err)
		}
	} else {
		envs = make([]*ChefEnvironment, 0, len(envNames))
		for _, e := range envNames {
			eo, _ := Get(e)
			if eo != nil {
				envs = append(envs, eo)
			}
		}
	}

	return envs, nil
}

// MakeDefaultEnvironment creates the default environment on startup.
func MakeDefaultEnvironment() {
	var de *ChefEnvironment
	if config.UsingDB() {
		// The default environment is pre-created in the db schema when
		// it's loaded. Re-indexing the default environment doesn't
		// hurt anything though, so just get the usual default env and
		// index it, not bothering with these other steps that are
		// easier to do with the in-memory mode.
		de = defaultEnvironment()
	} else {
		ds := datastore.New()
		// only create the new default environment if we don't already have one
		// saved
		if _, found := ds.Get("env", "_default"); found {
			return
		}
		de = defaultEnvironment()
		ds.Set("env", de.Name, de)
	}
	indexer.IndexObj(de)
}

func defaultEnvironment() *ChefEnvironment {
	return &ChefEnvironment{
		Name:             "_default",
		ChefType:         "environment",
		JSONClass:        "Chef::Environment",
		Description:      "The default Chef environment",
		Default:          map[string]interface{}{},
		Override:         map[string]interface{}{},
		CookbookVersions: map[string]string{},
	}
}

// Save the environment. Returns an error if you try to save the "_default"
// environment.
func (e *ChefEnvironment) Save() util.Gerror {
	if e.Name == "_default" {
		err := util.Errorf("The '_default' environment cannot be modified.")
		err.SetStatus(http.StatusMethodNotAllowed)
		return err
	}
	if config.Config.UseMySQL {
		err := e.saveEnvironmentMySQL()
		if err != nil {
			return err
		}
	} else if config.Config.UsePostgreSQL {
		err := e.saveEnvironmentPostgreSQL()
		if err != nil {
			return err
		}
	} else {
		ds := datastore.New()
		ds.Set("env", e.Name, e)
	}
	indexer.IndexObj(e)
	return nil
}

// Delete the environment, returning an error if you try to delete the
// "_default" environment.
func (e *ChefEnvironment) Delete() error {
	if e.Name == "_default" {
		err := fmt.Errorf("The '_default' environment cannot be modified.")
		return err
	}
	if config.UsingDB() {
		if err := e.deleteEnvironmentSQL(); err != nil {
			return nil
		}
	} else {
		ds := datastore.New()
		ds.Delete("env", e.Name)
	}
	indexer.DeleteItemFromCollection("environment", e.Name)
	return nil
}

// GetList gets a list of all environments on this server.
func GetList() []string {
	var envList []string
	if config.UsingDB() {
		envList = getEnvironmentList()
	} else {
		ds := datastore.New()
		envList = ds.GetList("env")
		envList = append(envList, "_default")
	}
	return envList
}

// GetName returns the environment's name.
func (e *ChefEnvironment) GetName() string {
	return e.Name
}

// URLType returns the base of an environment's URL.
func (e *ChefEnvironment) URLType() string {
	return "environments"
}

func (e *ChefEnvironment) cookbookList() ([]*cookbook.Cookbook, error) {
	return cookbook.AllCookbooks()
}

// AllCookbookHash returns a hash of the cookbooks and their versions available
// to this environment.
func (e *ChefEnvironment) AllCookbookHash(numVersions interface{}) map[string]interface{} {
	cbHash := make(map[string]interface{})
	cbList, _ := e.cookbookList()
	for _, cb := range cbList {
		if cb == nil {
			continue
		}
		cbHash[cb.Name] = cb.ConstrainedInfoHash(numVersions, e.CookbookVersions[cb.Name])
	}
	return cbHash
}

// RecipeList gets a list of recipes available to this environment.
func (e *ChefEnvironment) RecipeList() []string {
	recipeList := make(map[string]string)
	cbList, _ := e.cookbookList()
	for _, cb := range cbList {
		if cb == nil {
			continue
		}
		cbv := cb.LatestConstrained(e.CookbookVersions[cb.Name])
		if cbv == nil {
			continue
		}
		rlist, _ := cbv.RecipeList()

		for _, recipe := range rlist {
			recipeList[recipe] = recipe
		}
	}
	sortedRecipes := make([]string, len(recipeList))
	i := 0
	for k := range recipeList {
		sortedRecipes[i] = k
		i++
	}
	sort.Strings(sortedRecipes)
	return sortedRecipes
}

/* Search indexing methods */

// DocID returns the environment's name.
func (e *ChefEnvironment) DocID() string {
	return e.Name
}

// Index returns the environment's type so the indexer knows where it should go.
func (e *ChefEnvironment) Index() string {
	return "environment"
}

// Flatten the environment so it's suitable for indexing.
func (e *ChefEnvironment) Flatten() map[string]interface{} {
	return util.FlattenObj(e)
}

// AllEnvironments returns a slice of all environments on this server.
func AllEnvironments() []*ChefEnvironment {
	var environments []*ChefEnvironment
	if config.UsingDB() {
		environments = allEnvironmentsSQL()
	} else {
		envList := GetList()
		for _, e := range envList {
			en, err := Get(e)
			if err != nil {
				continue
			}
			environments = append(environments, en)
		}
	}
	return environments
}
