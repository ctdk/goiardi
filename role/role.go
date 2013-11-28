/* Roles, an important building block of Chef. */

/*
 * Copyright (c) 2013, Jeremy Bingham (<jbingham@gmail.com>)
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

package role

import (
	"github.com/ctdk/goiardi/data_store"
	"fmt"
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

func New(name string) (*Role, error){
	ds := data_store.New()
	if _, found := ds.Get("role", name); found {
		err := fmt.Errorf("Role %s already exists", name)
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

func NewFromJson(json_role map[string]interface{}) (*Role, error){
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

func (r *Role) UpdateFromJson(json_role map[string]interface{}) error {
	/* TODO - this and node.UpdateFromJson may be generalizeable with
	 * reflect - look into it. */
	if r.Name != json_role["name"] {
		err := fmt.Errorf("Role name %s and %s from JSON do not match.", r.Name, json_role["name"])
		return err
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
	ds := data_store.New()
	role, found := ds.Get("role", role_name)
	if !found {
		err := fmt.Errorf("Cannot load role %s", role_name)
		return nil, err
	}
	return role.(*Role), nil
}

func (r *Role) Save() error {
	ds := data_store.New()
	ds.Set("role", r.Name, r)
	return nil
}

func (r *Role) Delete() error {
	ds := data_store.New()
	ds.Delete("role", r.Name)
	return nil
}

func GetList() []string {
	ds := data_store.New()
	role_list := ds.GetList("role")
	return role_list
}

func (r *Role) GetName() string {
	return r.Name
}

func (r *Role) URLType() string {
	return "roles"
}
