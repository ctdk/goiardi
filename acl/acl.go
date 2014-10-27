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

package acl

import (
	"github.com/ctdk/goiardi/actor"
	"github.com/ctdk/goiardi/group"
)

var DefaultACLs = [5]string{
	"create",
	"read",
	"update",
	"delete",
	"grant",
}

type ACLOwner interface {
}

type ACLitem struct {
	Perm   string
	Actors []actor.Actor
	Groups []*group.Group
}

type ACL struct {
	ACLitems map[string]*ACLitem
	Owner    ACLOwner
}

func defaultACL(kind string, subkind string) *ACL {
	acl := make(ACL)
	// almost always we'd want these default acls
	acl.ACLitems = make(map[string]*ACLitem)
	for _, a := range DefaultACLs {
		acl.ACLitems[a] = &ACLitem{Perm: a}
	}
	switch kind {
	case "containers":
		switch subkind {
		case "$$root$$", "containers", "groups":
		case "cookbooks", "environments", "roles":
		// bit confusing here: chef-zero says cookbooks have both the
		// above and below defaults. Using the above for now.
		case "data":
		case "nodes":
		case "clients":
		case "sandboxes":
		default:
			// blank out the previous work
			acl = new(ACL)
		}
	case "groups":
		switch subkind {
		case "admins", "clients", "users":
		case "billing-admins":
		default:
			acl = new(ACL)
		}
	}
	return acl
}
