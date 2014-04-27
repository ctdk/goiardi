/* Roles, an important building block of Chef. */

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

// Package role provides roles, which are a way to share common attributes and
// run lists between different nodes.
package role

import (
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/data_store"
	"github.com/ctdk/goiardi/util"
	"github.com/ctdk/goiardi/indexer"
	"fmt"
	"net/http"
	"database/sql"
	"log"
)

/* Need env_run_lists?!!? */

type Role struct {
	Name string `json:"name"`
	ChefType string `json:"chef_type"`
	JsonClass string `json:"json_class"`
	RunList []string `json:"run_list"`
	EnvRunLists map[string][]string `json:"env_run_lists"`
	Description string `json:"description"`
	Default map[string]interface{} `json:"default_attributes"`
	Override map[string]interface{} `json:"override_attributes"`
}

func New(name string) (*Role, util.Gerror){
	var found bool
	if config.Config.UseMySQL {
		var err error
		found, err = checkForRoleMySQL(data_store.Dbh, name)
		_, err := data_store.CheckForOne(data_store.Dbh, "roles", name)
		if err != nil {
			gerr := util.Errorf(err.Error())
			gerr.SetStatus(http.StatusInternalServerError)
			return nil, gerr
		}
	} else {
		ds := data_store.New()
		_, found := ds.Get("role", name)
	}
	if found {
		err := util.Errorf("Role %s already exists", name)
		err.SetStatus(http.StatusConflict)
		return nil, err
	}
	if !util.ValidateDBagName(name){
		err := util.Errorf("Field 'name' invalid")
		err.SetStatus(http.StatusBadRequest)
		return nil, err
	}
	role := &Role{
		Name: name,
		ChefType: "role",
		JsonClass: "Chef::Role",
		RunList: []string{},
		EnvRunLists: map[string][]string{},
		Default: map[string]interface{}{},
		Override: map[string]interface{}{},
	}

	return role, nil
}

// Create a new role from the uploaded JSON.
func NewFromJson(json_role map[string]interface{}) (*Role, util.Gerror){
	role, err := New(json_role["name"].(string))
	if err != nil {
		return nil, err
	}
	err = role.UpdateFromJson(json_role)
	if err != nil {
		return nil, err
	}
	return role, nil
}

// Update an existing role with the uploaded JSON.
func (r *Role) UpdateFromJson(json_role map[string]interface{}) util.Gerror {
	/* TODO - this and node.UpdateFromJson may be generalizeable with
	 * reflect - look into it. */
	if r.Name != json_role["name"] {
		err := util.Errorf("Role name %s and %s from JSON do not match.", r.Name, json_role["name"])
		return err
	}

	/* Validations */

	/* Look for invalid top level elements. See node/node.go for more
	 * information. */
	valid_elements := []string{ "name", "json_class", "chef_type", "run_list", "env_run_lists", "default_attributes", "override_attributes", "description" }
	ValidElem:
	for k := range json_role {
		for _, i := range valid_elements {
			if k == i {
				continue ValidElem
			}
		}
		err := util.Errorf("Invalid key %s in request body", k)
		return err
	}

	var verr util.Gerror
	if json_role["run_list"], verr = util.ValidateRunList(json_role["run_list"]); verr != nil {
		return verr
	}

	if _, erl_exists := json_role["env_run_lists"]; erl_exists { 
		for k, v := range json_role["env_run_lists"].(map[string][]string) {
			if json_role["env_run_lists"].(map[string][]string)[k], verr = util.ValidateRunList(v); verr != nil {
				return verr
			}
		}
	} else {
		json_role["env_run_lists"] = make(map[string][]string)
	}

	attrs := []string{ "default_attributes", "override_attributes" }
	for _, a := range attrs {
		json_role[a], verr = util.ValidateAttributes(a, json_role[a])
		if verr != nil {
			return verr
		}
	}

	json_role["json_class"], verr = util.ValidateAsFieldString(json_role["json_class"])
	if verr != nil {
		if verr.Error() == "Field 'name' nil" {
			json_role["json_class"] = r.JsonClass
		} else {
			return verr
		}
	} else {
		if json_role["json_class"].(string) != "Chef::Role" {
			verr = util.Errorf("Field 'json_class' invalid")
			return verr
		}
	}

	// Roles can be empty, just force it into being a string
	json_role["description"], _ = util.ValidateAsString(json_role["description"])

	json_role["chef_type"], verr = util.ValidateAsFieldString(json_role["chef_type"])
	if verr != nil {
		if verr.Error() == "Field 'name' nil" {
			json_role["chef_type"] = r.ChefType
		} else {
			return verr
		}
	} else {
		if json_role["chef_type"].(string) != "role" {
			verr = util.Errorf("Field 'chef_type' invalid")
			return verr
		}
	}

	r.ChefType = json_role["chef_type"].(string)
	r.JsonClass = json_role["json_class"].(string)
	r.Description = json_role["description"].(string)
	r.RunList = json_role["run_list"].([]string)
	r.EnvRunLists = json_role["env_run_lists"].(map[string][]string)
	r.Default = json_role["default_attributes"].(map[string]interface{})
	r.Override = json_role["override_attributes"].(map[string]interface{})
	return nil
}


func Get(role_name string) (*Role, error){
	var role *Role
	var found bool
	if config.Config.UseMySQL {
		var err error
		role, err = getMySQL(role_name)
		if err != nil {
			if err == sql.ErrNoRows {
				found = false
			} else {
				return nil, err
			}
		} else {
			found = true
		}
	} else {
		ds := data_store.New()
		var r interface{}
		r, found = ds.Get("role", role_name)
		role = r.(*Role)
	}
	if !found {
		err := fmt.Errorf("Cannot load role %s", role_name)
		return nil, err
	}
	return role, nil
}

func (r *Role) Save() error {
	if config.Config.UseMySQL {
		if err := r.saveMySQL(); err != nil {
			return nil
		}
	} else {
		ds := data_store.New()
		ds.Set("role", r.Name, r)
	}
	indexer.IndexObj(r)
	return nil
}

func (r *Role) Delete() error {
	if config.Config.UseMySQL {
		if err := r.deleteMySQL(); err != nil {
			return err
		}
	} else {
		ds := data_store.New()
		ds.Delete("role", r.Name)
	}
	indexer.DeleteItemFromCollection("role", r.Name)
	return nil
}

// Get a list of the roles on this server.
func GetList() []string {
	var role_list []string
	if config.Config.UseMySQL {
		role_list = getListMySQL()
	} else {
		ds := data_store.New()
		role_list = ds.GetList("role")
	}
	return role_list
}

func (r *Role) GetName() string {
	return r.Name
}

func (r *Role) URLType() string {
	return "roles"
}

func (r *Role) DocId() string {
	return r.Name
}

func (r *Role) Index() string {
	return "role"
}

func (r *Role) Flatten() []string {
	flatten := util.FlattenObj(r)
	indexified := util.Indexify(flatten)
	return indexified
}
