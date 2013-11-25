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
	"fmt"
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

func New(name string) (*Node, error) {
	/* check for an existing node with this name */
	ds := data_store.New()
	if _, found := ds.Get("node", name); found {
		err := fmt.Errorf("Node %s already exists", name)
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

func NewFromJson(json_node map[string]interface{}) (*Node, error){
	node, err := New(json_node["name"].(string))
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
		err := fmt.Errorf("Node %s not found", node_name)
		return nil, err
	}
	return node.(*Node), nil
}

func (n *Node) UpdateFromJson(json_node map[string]interface{}) error {
	/* It's actually totally legitimate to save a node with a different
	 * name than you started with, but we need to get/create a new node for
	 * it is all. */
	if n.Name != json_node["name"] {
		err := fmt.Errorf("Node name %s and %s from JSON do not match.", n.Name, json_node["name"])
		return err
	}
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
	return nil
}

func (n *Node) Delete() error {
	ds := data_store.New()
	ds.Delete("node", n.Name)
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
