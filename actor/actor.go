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

// Package actor implements actors, which is an interface encompassing both
// clients or users. They serve many of the same functions and formerly were
// implemented using the same object, but are now different types. Thus,Actor
// is now an interface rather than being a distinct type of object encompassing
// both.
package actor

import (
	"github.com/ctdk/goiardi/client"
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/organization"
	"github.com/ctdk/goiardi/user"
	"github.com/ctdk/goiardi/util"
	"net/http"
)

// Actor is an interface for objects that can make requests to the server.
type Actor interface {
	IsAdmin() bool
	IsValidator() bool
	IsSelf(interface{}) bool
	IsUser() bool
	IsClient() bool
	PublicKey() string
	SetPublicKey(interface{}) error
	GetName() string
	CheckPermEdit(map[string]interface{}, string) util.Gerror
	OrgName() string
	ACLName() string
	Authz() string
	IsACLRole() bool
	GetId() int64
}

// GetReqUser gets the actor making the request. If use-auth is not on, always
// returns the admin user.
func GetReqUser(org *organization.Organization, name string) (Actor, util.Gerror) {
	/* If UseAuth is turned off, use the automatically created admin user */
	if !config.Config.UseAuth {
		name = "admin"
	}
	var c Actor
	var err error
	if org != nil {
		c, err = client.Get(org, name)
	}
	if err != nil || org == nil {
		/* Theoretically it should be hard to reach this point, since
		 * if the signed request was accepted the user ought to exist.
		 * Still, best to be cautious. */
		// TODO: check that the user in question has rights to this
		// organization
		u, cerr := user.Get(name)
		if cerr != nil {
			var errmsg string
			if err != nil {
				errmsg = err.Error()
			}
			gerr := util.Errorf("Neither a client nor a user named '%s' could be found. In addition, the following errors were reported: %s -- %s", name, errmsg, cerr.Error())
			gerr.SetStatus(http.StatusUnauthorized)
			return nil, gerr
		}
		c = u
	}
	return c, nil
}

// Right now it just wraps GetReqUser, but one or the other may change
// drastically in the future.
func GetActor(org *organization.Organization, name string) (Actor, util.Gerror) {
	return GetReqUser(org, name)
}
