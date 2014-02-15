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

package search

import (
	"testing"
	"github.com/ctdk/goiardi/node"
	"github.com/ctdk/goiardi/role"
	"github.com/ctdk/goiardi/environment"
	"github.com/ctdk/goiardi/actor"
	"github.com/ctdk/goiardi/data_bag"
	"fmt"
)

// Most search testing can be handled fine with chef-pedant, but that's no
// reason to not have some go tests for it too.
var node1 *node.Node
var node2 *node.Node
var node3 *node.Node
var node4 *node.Node
var role1 *role.Role
var role2 *role.Role
var role3 *role.Role
var role4 *role.Role
var env1 *environment.ChefEnvironment
var env2 *environment.ChefEnvironment
var env3 *environment.ChefEnvironment
var env4 *environment.ChefEnvironment
var client1 *actor.Actor
var client2 *actor.Actor
var client3 *actor.Actor
var client4 *actor.Actor
var dbag1 *data_bag.DataBag
var dbag2 *data_bag.DataBag
var dbag3 *data_bag.DataBag
var dbag4 *data_bag.DataBag

func makeSearchItems() int{
	/* Gotta populate the search index */
	nodes := make([]*node.Node, 4)
	roles := make([]*role.Role, 4)
	envs := make([]*environment.ChefEnvironment, 4)
	clients := make([]*actor.Actor, 4)
	dbags := make([]*data_bag.DataBag, 4)

	for i := 0; i < 4; i++ {
		nodes[i] = node.New(
	}

	/* Make this function return something so the compiler's happy building
	 * the tests. */
	return 1
}

var v = makeSearchItems()

func TestFoo(t *testing.T){
	return
}
