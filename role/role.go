/* Roles, an important building block of Chef. */

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

// Package role provides roles, which are a way to share common attributes and
// run lists between different nodes.
package role

import (
	"database/sql"
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/datastore"
	"github.com/ctdk/goiardi/indexer"
	"github.com/ctdk/goiardi/organization"
	"github.com/ctdk/goiardi/util"
	"net/http"
)

/* Need env_run_lists?!!? */

// Role is a way to specify shared run lists and attributes for nodes.
type Role struct {
	Name        string                 `json:"name"`
	ChefType    string                 `json:"chef_type"`
	JSONClass   string                 `json:"json_class"`
	RunList     []string               `json:"run_list"`
	EnvRunLists map[string][]string    `json:"env_run_lists"`
	Description string                 `json:"description"`
	Default     map[string]interface{} `json:"default_attributes"`
	Override    map[string]interface{} `json:"override_attributes"`
	org         *organization.Organization
}

// New creates a new role.
func New(org *organization.Organization, name string) (*Role, util.Gerror) {
	var found bool
	if config.UsingDB() {
		var err error
		found, err = checkForRoleSQL(datastore.Dbh, org, name)
		if err != nil {
			gerr := util.Errorf(err.Error())
			gerr.SetStatus(http.StatusInternalServerError)
			return nil, gerr
		}
	} else {
		ds := datastore.New()
		_, found = ds.Get(org.DataKey("role"), name)
	}
	if found {
		err := util.Errorf("Role %s already exists", name)
		err.SetStatus(http.StatusConflict)
		return nil, err
	}
	if !util.ValidateDBagName(name) {
		err := util.Errorf("Field 'name' invalid")
		err.SetStatus(http.StatusBadRequest)
		return nil, err
	}
	role := &Role{
		Name:        name,
		ChefType:    "role",
		JSONClass:   "Chef::Role",
		RunList:     []string{},
		EnvRunLists: map[string][]string{},
		Default:     map[string]interface{}{},
		Override:    map[string]interface{}{},
		org:         org,
	}

	return role, nil
}

// NewFromJSON creates a new role from the uploaded JSON.
func NewFromJSON(org *organization.Organization, jsonRole map[string]interface{}) (*Role, util.Gerror) {
	role, err := New(org, jsonRole["name"].(string))
	if err != nil {
		return nil, err
	}
	err = role.UpdateFromJSON(jsonRole)
	if err != nil {
		return nil, err
	}
	return role, nil
}

// UpdateFromJSON updates an existing role with the uploaded JSON.
func (r *Role) UpdateFromJSON(jsonRole map[string]interface{}) util.Gerror {
	/* TODO - this and node.UpdateFromJSON may be generalizeable with
	 * reflect - look into it. */
	if r.Name != jsonRole["name"] {
		err := util.Errorf("Role name %s and %s from JSON do not match.", r.Name, jsonRole["name"])
		return err
	}

	/* Validations */

	/* Look for invalid top level elements. See node/node.go for more
	 * information. */
	validElements := []string{"name", "json_class", "chef_type", "run_list", "env_run_lists", "default_attributes", "override_attributes", "description"}
ValidElem:
	for k := range jsonRole {
		for _, i := range validElements {
			if k == i {
				continue ValidElem
			}
		}
		err := util.Errorf("Invalid key %s in request body", k)
		return err
	}

	var verr util.Gerror
	if jsonRole["run_list"], verr = util.ValidateRunList(jsonRole["run_list"]); verr != nil {
		return verr
	}

	if _, erlExists := jsonRole["env_run_lists"]; erlExists {
		for k, v := range jsonRole["env_run_lists"].(map[string][]string) {
			if jsonRole["env_run_lists"].(map[string][]string)[k], verr = util.ValidateRunList(v); verr != nil {
				return verr
			}
		}
	} else {
		jsonRole["env_run_lists"] = make(map[string][]string)
	}

	attrs := []string{"default_attributes", "override_attributes"}
	for _, a := range attrs {
		jsonRole[a], verr = util.ValidateAttributes(a, jsonRole[a])
		if verr != nil {
			return verr
		}
	}

	jsonRole["json_class"], verr = util.ValidateAsFieldString(jsonRole["json_class"])
	if verr != nil {
		if verr.Error() == "Field 'name' nil" {
			jsonRole["json_class"] = r.JSONClass
		} else {
			return verr
		}
	} else {
		if jsonRole["json_class"].(string) != "Chef::Role" {
			verr = util.Errorf("Field 'json_class' invalid")
			return verr
		}
	}

	// Roles can be empty, just force it into being a string
	jsonRole["description"], _ = util.ValidateAsString(jsonRole["description"])

	jsonRole["chef_type"], verr = util.ValidateAsFieldString(jsonRole["chef_type"])
	if verr != nil {
		if verr.Error() == "Field 'name' nil" {
			jsonRole["chef_type"] = r.ChefType
		} else {
			return verr
		}
	} else {
		if jsonRole["chef_type"].(string) != "role" {
			verr = util.Errorf("Field 'chef_type' invalid")
			return verr
		}
	}

	r.ChefType = jsonRole["chef_type"].(string)
	r.JSONClass = jsonRole["json_class"].(string)
	r.Description = jsonRole["description"].(string)
	r.RunList = jsonRole["run_list"].([]string)
	r.EnvRunLists = jsonRole["env_run_lists"].(map[string][]string)
	r.Default = jsonRole["default_attributes"].(map[string]interface{})
	r.Override = jsonRole["override_attributes"].(map[string]interface{})
	return nil
}

// Get a role.
func Get(org *organization.Organization, roleName string) (*Role, util.Gerror) {
	var role *Role
	var found bool
	if config.UsingDB() {
		var err error
		role, err = getSQL(roleName)
		if err != nil {
			if err == sql.ErrNoRows {
				found = false
			} else {
				return nil, util.CastErr(err)
			}
		} else {
			found = true
		}
	} else {
		ds := datastore.New()
		var r interface{}
		r, found = ds.Get(org.DataKey("role"), roleName)
		if r != nil {
			role = r.(*Role)
			role.org = org
		}
	}
	if !found {
		err := util.Errorf("Cannot load role %s", roleName)
		err.SetStatus(http.StatusNotFound)
		return nil, err
	}
	return role, nil
}

// DoesExist checks if this role exists or not
func DoesExist(org *organization.Organization, roleName string) (bool, util.Gerror) {
	var found bool
	if config.UsingDB() {
		var cerr error
		found, cerr = checkForRoleSQL(datastore.Dbh, org, roleName)
		if cerr != nil {
			err := util.Errorf(cerr.Error())
			err.SetStatus(http.StatusInternalServerError)
			return false, err
		}
	} else {
		ds := datastore.New()
		_, found = ds.Get(org.DataKey("role"), roleName)
	}
	return found, nil
}

// GetMulti gets multiple roles from a slice of role names.
func GetMulti(org *organization.Organization, roleNames []string) ([]*Role, util.Gerror) {
	var roles []*Role
	if config.UsingDB() {
		var err error
		roles, err = getMultiSQL(roleNames)
		if err != nil && err != sql.ErrNoRows {
			return nil, util.CastErr(err)
		}
	} else {
		roles = make([]*Role, 0, len(roleNames))
		for _, r := range roleNames {
			ro, _ := Get(org, r)
			if ro != nil {
				roles = append(roles, ro)
			}
		}
	}

	return roles, nil
}

// Save the role.
func (r *Role) Save() util.Gerror {
	if config.Config.UseMySQL {
		if err := r.saveMySQL(); err != nil {
			return util.CastErr(err)
		}
	} else if config.Config.UsePostgreSQL {
		if err := r.savePostgreSQL(); err != nil {
			return util.CastErr(err)
		}
	} else {
		ds := datastore.New()
		ds.Set(r.org.DataKey("role"), r.Name, r)
	}
	indexer.IndexObj(r)
	return nil
}

// Delete a role.
func (r *Role) Delete() util.Gerror {
	if config.UsingDB() {
		if err := r.deleteSQL(); err != nil {
			return util.CastErr(err)
		}
	} else {
		ds := datastore.New()
		ds.Delete(r.org.DataKey("role"), r.Name)
	}
	indexer.DeleteItemFromCollection(r.org.Name, "role", r.Name)
	_, aerr := r.org.PermCheck.DeleteItemACL(r)
	if aerr != nil {
		return util.CastErr(aerr)
	}
	return nil
}

// GetList gets a list of the roles on this server.
func GetList(org *organization.Organization) []string {
	var roleList []string
	if config.UsingDB() {
		roleList = getListSQL()
	} else {
		ds := datastore.New()
		roleList = ds.GetList(org.DataKey("role"))
	}
	return roleList
}

// GetName returns the role's name.
func (r *Role) GetName() string {
	return r.Name
}

// URLType returns the base element of a role's URL.
func (r *Role) URLType() string {
	return "roles"
}

func (r *Role) ContainerType() string {
	return r.URLType()
}

func (r *Role) ContainerKind() string {
	return "containers"
}

// OrgName returns the organization this role belongs to.
func (r *Role) OrgName() string {
	return r.org.Name
}

// DocID returns the role's name.
func (r *Role) DocID() string {
	return r.Name
}

// Index tells the indexer where it should put the role when it's being indexed.
func (r *Role) Index() string {
	return "role"
}

// Flatten a role so it's suitable for indexing.
func (r *Role) Flatten() map[string]interface{} {
	return util.FlattenObj(r)
}

// AllRoles returns all the roles on the server
func AllRoles(org *organization.Organization) []*Role {
	var roles []*Role
	if config.UsingDB() {
		roles = allRolesSQL()
	} else {
		roleList := GetList(org)
		roles = make([]*Role, 0, len(roleList))
		for _, r := range roleList {
			ro, err := Get(org, r)
			if err != nil {
				continue
			}
			roles = append(roles, ro)
		}
	}
	return roles
}
