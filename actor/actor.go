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

// Package actor implements actors, which is an interface encompassing both
// clients or users. They serve many of the same functions and formerly were
// implemented using the same object, but are now different types. Thus,Actor
// is now an interface rather than being a distinct type of object encompassing
// both.
package actor

import (
	"github.com/ctdk/goiardi/client"
	"github.com/ctdk/goiardi/config"
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
}

// GetReqUser gets the actor making the request. If use-auth is not on, always 
// returns the admin user.
func GetReqUser(name string) (Actor, util.Gerror) {
	/* If UseAuth is turned off, use the automatically created admin user */
	if !config.Config.UseAuth {
		name = "admin"
	}
	var c Actor
	var err error
	c, err = client.Get(name)
	if err != nil {
		/* Theoretically it should be hard to reach this point, since
		 * if the signed request was accepted the user ought to exist.
		 * Still, best to be cautious. */
		u, cerr := user.Get(name)
		if cerr != nil {
			gerr := util.Errorf("Neither a client nor a user named '%s' could be found. In addition, the following errors were reported: %s -- %s", name, err.Error(), cerr.Error())
			gerr.SetStatus(http.StatusUnauthorized)
			return nil, gerr
		}
		c = u
	}
	return c, nil
}
