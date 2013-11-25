/* Environments. */

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

package environment

import (
	"github.com/ctdk/goiardi/data_store"
	"github.com/ctdk/goiardi/cookbook"
	"fmt"
	"sort"
)

type ChefEnvironment struct {
	Name string `json:"name"`
	ChefType string `json:"chef_type"`
	JsonClass string `json:"json_class"`
	Description string `json:"description"`
	Default map[string]interface{} `json:"default_attributes"`
	Override map[string]interface{} `json:"override_attributes"`
	CookbookVersions map[string]string `json:"cookbook_versions"`
}

func New(name string) (*ChefEnvironment, error){
	ds := data_store.New()
	if _, found := ds.Get("env", name); found || name == "_default" {
		err := fmt.Errorf("Environment %s already exists.", name)
		return nil, err
	}
	env := &ChefEnvironment{
		Name: name,
		ChefType: "environment",
		JsonClass: "Chef::Environment",
		Default: map[string]interface{}{},
		Override: map[string]interface{}{},
		CookbookVersions: map[string]string{},
	}
	return env, nil
}

func NewFromJson(json_env map[string]interface{}) (*ChefEnvironment, error){
	env, err := New(json_env["name"].(string))
	if err != nil {
		return nil, err
	}
	err = env.UpdateFromJson(json_env)
	if err != nil {
		return nil, err
	}
	return env, nil
}

func (e *ChefEnvironment)UpdateFromJson(json_env map[string]interface{}) error {
	if e.Name != json_env["name"].(string) {
		err := fmt.Errorf("Environment name %s and %s from JSON do not match", e.Name, json_env["name"].(string))
		return err
	} else if e.Name == "_default" {
		err := fmt.Errorf("Default environment cannot be modified.")
		return err
	}
	e.ChefType = json_env["chef_type"].(string)
	e.JsonClass = json_env["json_class"].(string)
	e.Description = json_env["description"].(string)
	e.Default = json_env["default_attributes"].(map[string]interface{})
	e.Override = json_env["override_attributes"].(map[string]interface{})
	/* loop over the cookbook versions */
	for c, v := range json_env["cookbook_versions"].(map[string]interface{}){
		e.CookbookVersions[c] = v.(string)
	}

	return nil
}

func Get(env_name string) (*ChefEnvironment, error){
	if env_name == "_default" {
		return defaultEnvironment(), nil
	}
	ds := data_store.New()
	env, found := ds.Get("env", env_name)
	if !found {
		err := fmt.Errorf("Environment %s not found", env_name)
		return nil, err
	}
	return env.(*ChefEnvironment), nil
}

func defaultEnvironment() (*ChefEnvironment) {
	return &ChefEnvironment{
		Name: "_default",
		ChefType: "environment",
		JsonClass: "Chef::Environment",
		Description: "The default environment",
		Default: map[string]interface{}{},
		Override: map[string]interface{}{},
		CookbookVersions: map[string]string{},
	}
}

func (e *ChefEnvironment) Save() error {
	if e.Name == "_default" {
		err := fmt.Errorf("Default environment cannot be modified.")
		return err
	}
	ds := data_store.New()
	ds.Set("env", e.Name, e)
	return nil
}

func (e *ChefEnvironment) Delete() error {
	if e.Name == "_default" {
		err := fmt.Errorf("Default environment cannot be deleted.")
		return err
	}
	ds := data_store.New()
	ds.Delete("env", e.Name)
	return nil
}

func GetList() []string {
	ds := data_store.New()
	env_list := ds.GetList("env")
	env_list = append(env_list, "_default")
	return env_list
}

func (e *ChefEnvironment) GetName() string {
	return e.Name
}

func (e *ChefEnvironment) URLType() string {
	return "environments"
}

func (e *ChefEnvironment) cookbookList() []*cookbook.Cookbook {
	cb_list := cookbook.GetList()
	cookbooks := make([]*cookbook.Cookbook, len(cb_list))
	for i, cb := range cb_list {
		cookbooks[i], _ = cookbook.Get(cb)
	}
	return cookbooks
}

func (e *ChefEnvironment) AllCookbookHash(num_versions interface{}) map[string]interface{} {
	cb_hash := make(map[string]interface{})
	cb_list := e.cookbookList()
	for _, cb := range cb_list {
		if cb == nil {
			continue
		}
		cb_hash[cb.Name] = cb.ConstrainedInfoHash(num_versions, e.CookbookVersions[cb.Name])
	}
	return cb_hash
}

func (e *ChefEnvironment) RecipeList() []string {
	recipe_list := make(map[string]string)
	cb_list := e.cookbookList()
	for _, cb := range cb_list {
		if cb == nil {
			continue
		}
		cbv := cb.LatestConstrained(e.CookbookVersions[cb.Name])
		if cbv == nil {
			continue
		}
		for _, recipe := range cbv.RecipeList() {
			recipe_list[recipe] = recipe
		}
	}
	sorted_recipes := make([]string, len(recipe_list))
	i := 0
	for k := range recipe_list {
		sorted_recipes[i] = k
		i++
	}
	sort.Strings(sorted_recipes)
	return sorted_recipes
}
