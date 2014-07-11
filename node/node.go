/* Node object/class/whatever it is that Go calls them. */

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

// Package node implements nodes. They do chef-client runs.
package node

import (
	"database/sql"
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/datastore"
	"github.com/ctdk/goiardi/indexer"
	"github.com/ctdk/goiardi/util"
	"net/http"
)

// Node is a basic Chef node, holding the run list and attributes of the node.
type Node struct {
	Name            string                 `json:"name"`
	ChefEnvironment string                 `json:"chef_environment"`
	RunList         []string               `json:"run_list"`
	JSONClass       string                 `json:"json_class"`
	ChefType        string                 `json:"chef_type"`
	Automatic       map[string]interface{} `json:"automatic"`
	Normal          map[string]interface{} `json:"normal"`
	Default         map[string]interface{} `json:"default"`
	Override        map[string]interface{} `json:"override"`
}

// New makes a new node.
func New(name string) (*Node, util.Gerror) {
	/* check for an existing node with this name */
	if !util.ValidateDBagName(name) {
		err := util.Errorf("Field 'name' invalid")
		return nil, err
	}

	var found bool
	if config.UsingDB() {
		// will need redone if orgs ever get implemented
		var err error
		found, err = checkForNodeSQL(datastore.Dbh, name)
		if err != nil {
			gerr := util.Errorf(err.Error())
			gerr.SetStatus(http.StatusInternalServerError)
			return nil, gerr
		}
	} else {
		ds := datastore.New()
		_, found = ds.Get("node", name)
	}
	if found {
		err := util.Errorf("Node %s already exists", name)
		err.SetStatus(http.StatusConflict)
		return nil, err
	}

	/* No node, we make a new one */
	node := &Node{
		Name:            name,
		ChefEnvironment: "_default",
		ChefType:        "node",
		JSONClass:       "Chef::Node",
		RunList:         []string{},
		Automatic:       map[string]interface{}{},
		Normal:          map[string]interface{}{},
		Default:         map[string]interface{}{},
		Override:        map[string]interface{}{},
	}
	return node, nil
}

// NewFromJSON creates a new node from the uploaded JSON.
func NewFromJSON(jsonNode map[string]interface{}) (*Node, util.Gerror) {
	nodeName, nerr := util.ValidateAsString(jsonNode["name"])
	if nerr != nil {
		return nil, nerr
	}
	node, err := New(nodeName)
	if err != nil {
		return nil, err
	}
	err = node.UpdateFromJSON(jsonNode)
	if err != nil {
		return nil, err
	}
	return node, nil
}

// Get a node.
func Get(nodeName string) (*Node, util.Gerror) {
	var node *Node
	var found bool
	if config.UsingDB() {
		var err error
		node, err = getSQL(nodeName)
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
		var n interface{}
		n, found = ds.Get("node", nodeName)
		if n != nil {
			node = n.(*Node)
		}
	}
	if !found {
		err := util.Errorf("node '%s' not found", nodeName)
		err.SetStatus(http.StatusNotFound)
		return nil, err
	}
	return node, nil
}

// UpdateFromJSON updates an existing node with the uploaded JSON.
func (n *Node) UpdateFromJSON(jsonNode map[string]interface{}) util.Gerror {
	/* It's actually totally legitimate to save a node with a different
	 * name than you started with, but we need to get/create a new node for
	 * it is all. */
	nodeName, nerr := util.ValidateAsString(jsonNode["name"])
	if nerr != nil {
		return nerr
	}
	if n.Name != nodeName {
		err := util.Errorf("Node name %s and %s from JSON do not match.", n.Name, nodeName)
		return err
	}

	/* Validations */

	/* Look for invalid top level elements. *We* don't have to worry about
		 * them, but chef-pedant cares (probably because Chef <=10 stores
	 	 * json objects directly, dunno about Chef 11). */
	validElements := []string{"name", "json_class", "chef_type", "chef_environment", "run_list", "override", "normal", "default", "automatic"}
ValidElem:
	for k := range jsonNode {
		for _, i := range validElements {
			if k == i {
				continue ValidElem
			}
		}
		err := util.Errorf("Invalid key %s in request body", k)
		return err
	}

	var verr util.Gerror
	jsonNode["run_list"], verr = util.ValidateRunList(jsonNode["run_list"])
	if verr != nil {
		return verr
	}
	attrs := []string{"normal", "automatic", "default", "override"}
	for _, a := range attrs {
		jsonNode[a], verr = util.ValidateAttributes(a, jsonNode[a])
		if verr != nil {
			return verr
		}
	}

	jsonNode["chef_environment"], verr = util.ValidateAsFieldString(jsonNode["chef_environment"])
	if verr != nil {
		if verr.Error() == "Field 'name' nil" {
			jsonNode["chef_environment"] = n.ChefEnvironment
		} else {
			return verr
		}
	} else {
		if !util.ValidateEnvName(jsonNode["chef_environment"].(string)) {
			verr = util.Errorf("Field 'chef_environment' invalid")
			return verr
		}
	}

	jsonNode["json_class"], verr = util.ValidateAsFieldString(jsonNode["json_class"])
	if verr != nil {
		if verr.Error() == "Field 'name' nil" {
			jsonNode["json_class"] = n.JSONClass
		} else {
			return verr
		}
	} else {
		if jsonNode["json_class"].(string) != "Chef::Node" {
			verr = util.Errorf("Field 'json_class' invalid")
			return verr
		}
	}

	jsonNode["chef_type"], verr = util.ValidateAsFieldString(jsonNode["chef_type"])
	if verr != nil {
		if verr.Error() == "Field 'name' nil" {
			jsonNode["chef_type"] = n.ChefType
		} else {
			return verr
		}
	} else {
		if jsonNode["chef_type"].(string) != "node" {
			verr = util.Errorf("Field 'chef_type' invalid")
			return verr
		}
	}

	/* and setting */
	n.ChefEnvironment = jsonNode["chef_environment"].(string)
	n.ChefType = jsonNode["chef_type"].(string)
	n.JSONClass = jsonNode["json_class"].(string)
	n.RunList = jsonNode["run_list"].([]string)
	n.Normal = jsonNode["normal"].(map[string]interface{})
	n.Automatic = jsonNode["automatic"].(map[string]interface{})
	n.Default = jsonNode["default"].(map[string]interface{})
	n.Override = jsonNode["override"].(map[string]interface{})
	return nil
}

// Save the node.
func (n *Node) Save() error {
	if config.UsingDB() {
		if err := n.saveSQL(); err != nil {
			return err
		}
	} else {
		ds := datastore.New()
		ds.Set("node", n.Name, n)
	}
	/* TODO Later: excellent candidate for a goroutine */
	indexer.IndexObj(n)
	return nil
}

// Delete the node.
func (n *Node) Delete() error {
	if config.UsingDB() {
		if err := n.deleteSQL(); err != nil {
			return err
		}
	} else {
		ds := datastore.New()
		ds.Delete("node", n.Name)
		// TODO: This may need a different config flag?
		if config.Config.UseSerf {
			n.deleteStatuses()
		}
	}
	indexer.DeleteItemFromCollection("node", n.Name)
	return nil
}

// GetList gets a list of the nodes on this server.
func GetList() []string {
	var nodeList []string
	if config.UsingDB() {
		nodeList = getListSQL()
	} else {
		ds := datastore.New()
		nodeList = ds.GetList("node")
	}
	return nodeList
}

// GetFromEnv returns all nodes that belong to the given environment.
func GetFromEnv(envName string) ([]*Node, error) {
	if config.UsingDB() {
		return getNodesInEnvSQL(envName)
	}
	var envNodes []*Node
	nodeList := GetList()
	for _, n := range nodeList {
		chefNode, _ := Get(n)
		if chefNode == nil {
			continue
		}
		if chefNode.ChefEnvironment == envName {
			envNodes = append(envNodes, chefNode)
		}
	}
	return envNodes, nil
}

// GetName returns the node's name.
func (n *Node) GetName() string {
	return n.Name
}

// URLType returns the base element of a node's URL.
func (n *Node) URLType() string {
	return "nodes"
}

/* Functions to support indexing */

// DocID returns the node's name.
func (n *Node) DocID() string {
	return n.Name
}

// Index tells the indexer where the node should go.
func (n *Node) Index() string {
	return "node"
}

// Flatten a node for indexing.
func (n *Node) Flatten() []string {
	flatten := util.FlattenObj(n)
	indexified := util.Indexify(flatten)
	return indexified
}

// AllNodes returns all the nodes on the server
func AllNodes() []*Node {
	var nodes []*Node
	if config.UsingDB() {
		nodes = allNodesSQL()
	} else {
		nodeList := GetList()
		for _, n := range nodeList {
			no, err := Get(n)
			if err != nil {
				continue
			}
			nodes = append(nodes, no)
		}
	}
	return nodes
}
