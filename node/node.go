/* Node object/class/whatever it is that Go calls them.
 * TODO: document what a node is. */

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

package node

import (
	"github.com/ctdk/goiardi/data_store"
	"github.com/ctdk/goiardi/util"
	"github.com/ctdk/goiardi/indexer"
	"fmt"
	"net/http"
)

type Node struct {
	Name string `json:"name"`
	ChefEnvironment string `json:"chef_environment"`
	RunList []string `json:"run_list"`
	JsonClass string `json:"json_class"`
	ChefType string `json:"chef_type"`
	Automatic map[string]interface{} `json:"automatic"`
	Normal map[string]interface{} `json:"normal"`
	Default map[string]interface{} `json:"default"`
	Override map[string]interface{} `json:"override"`
}

func New(name string) (*Node, util.Gerror) {
	/* check for an existing node with this name */
	ds := data_store.New()
	if _, found := ds.Get("node", name); found {
		err := util.Errorf("Node %s already exists", name)
		err.SetStatus(http.StatusConflict)
		return nil, err
	}
	if !util.ValidateDBagName(name){
		err := util.Errorf("Field 'name' invalid")
		return nil, err
	}
	/* No node, we make a new one */
	node := &Node{
		Name: name,
		ChefEnvironment: "_default",
		ChefType: "node",
		JsonClass: "Chef::Node",
		RunList: []string{},
		Automatic: map[string]interface{}{},
		Normal: map[string]interface{}{},
		Default: map[string]interface{}{},
		Override: map[string]interface{}{},
	}
	return node, nil
}

func NewFromJson(json_node map[string]interface{}) (*Node, util.Gerror){
	node_name, nerr := util.ValidateAsString(json_node["name"])
	if nerr != nil {
		return nil, nerr
	}
	node, err := New(node_name)
	if err != nil {
		return nil, err
	}
	err = node.UpdateFromJson(json_node)
	if err != nil {
		return nil, err
	}
	return node, nil
}

func Get(node_name string) (*Node, error) {
	ds := data_store.New()
	node, found := ds.Get("node", node_name)
	if !found {
		err := fmt.Errorf("node '%s' not found", node_name)
		return nil, err
	}
	return node.(*Node), nil
}

func (n *Node) UpdateFromJson(json_node map[string]interface{}) util.Gerror {
	/* It's actually totally legitimate to save a node with a different
	 * name than you started with, but we need to get/create a new node for
	 * it is all. */
	node_name, nerr := util.ValidateAsString(json_node["name"])
	if nerr != nil {
		return nerr
	}
	if n.Name != node_name {
		err := util.Errorf("Node name %s and %s from JSON do not match.", n.Name, node_name)
		return err
	}

	/* Validations */

	/* Look for invalid top level elements. *We* don't have to worry about
	 * them, but chef-pedant cares (probably because Chef <=10 stores
 	 * json objects directly, dunno about Chef 11). */
	valid_elements := []string{ "name", "json_class", "chef_type", "chef_environment", "run_list", "override", "normal", "default", "automatic" }
	ValidElem:
	for k, _ := range json_node {
		for _, i := range valid_elements {
			if k == i {
				continue ValidElem
			}
		}
		err := util.Errorf("Invalid key %s in request body", k)
		return err
	}

	var verr util.Gerror
	json_node["run_list"], verr = util.ValidateRunList(json_node["run_list"])
	if verr != nil {
		return verr
	}
	attrs := []string{ "normal", "automatic", "default", "override" }
	for _, a := range attrs {
		json_node[a], verr = util.ValidateAttributes(a, json_node[a])
		if verr != nil {
			return verr
		}
	}

	json_node["chef_environment"], verr = util.ValidateAsFieldString(json_node["chef_environment"])
	if verr != nil {
		if verr.Error() == "Field 'name' nil" {
			json_node["chef_environment"] = n.ChefEnvironment
		} else {
			return verr
		}
	} else {
		if !util.ValidateEnvName(json_node["chef_environment"].(string)) {
			verr = util.Errorf("Field 'chef_environment' invalid")
			return verr
		}
	}

	json_node["json_class"], verr = util.ValidateAsFieldString(json_node["json_class"])
	if verr != nil {
		if verr.Error() == "Field 'name' nil" {
			json_node["json_class"] = n.JsonClass
		} else {
			return verr
		}
	} else {
		if json_node["json_class"].(string) != "Chef::Node" {
			verr = util.Errorf("Field 'json_class' invalid")
			return verr
		}
	}


	json_node["chef_type"], verr = util.ValidateAsFieldString(json_node["chef_type"])
	if verr != nil {
		if verr.Error() == "Field 'name' nil" {
			json_node["chef_type"] = n.ChefType
		} else {
			return verr
		}
	} else {
		if json_node["chef_type"].(string) != "node" {
			verr = util.Errorf("Field 'chef_type' invalid")
			return verr
		}
	}

	/* and setting */
	n.ChefEnvironment = json_node["chef_environment"].(string)
	n.ChefType = json_node["chef_type"].(string)
	n.JsonClass = json_node["json_class"].(string)
	n.RunList = json_node["run_list"].([]string)
	n.Normal = json_node["normal"].(map[string]interface{})
	n.Automatic = json_node["automatic"].(map[string]interface{})
	n.Default = json_node["default"].(map[string]interface{})
	n.Override = json_node["override"].(map[string]interface{})
	return nil
}

func (n *Node) Save() error {
	ds := data_store.New()
	ds.Set("node", n.Name, n)
	/* TODO Later: excellent candidate for a goroutine */
	indexer.IndexObj(n)
	return nil
}

func (n *Node) Delete() error {
	ds := data_store.New()
	ds.Delete("node", n.Name)
	indexer.DeleteItemFromCollection("node", n.Name)
	return nil
}

func GetList() []string {
	ds := data_store.New()
	node_list := ds.GetList("node")
	return node_list
}

func (n *Node) GetName() string {
	return n.Name
}

func (n *Node) URLType() string {
	return "nodes"
}

/* Functions to support indexing */
func (n *Node) DocId() string {
	return n.Name
}

func (n *Node) Index() string {
	return "node"
}

func (n *Node) Flatten() []string {
	flatten := util.FlattenObj(n)
	indexified := util.Indexify(flatten)
	return indexified
}
