/*
 * Copyright (c) 2013-2017, Jeremy Bingham (<jeremy@goiardi.gl>)
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

// Package reqctx contains some types, variables, and functions for request
// contexts.
package reqctx

import (
	"context"
	"github.com/ctdk/goiardi/actor"
	"github.com/ctdk/goiardi/gerror"
	"github.com/ctdk/goiardi/organization"
)

// OpUserCtxKey is a string type for a key for setting and fetching the request
// user in the request's context.
type OpUserCtxKey string

// OrgCtxKey is a string type for setting and fetching the organization for a
// request.
type OrgCtxKey string

// OpUserKey is the default context key for the opUser stored in a request
// context.
var OpUserKey OpUserCtxKey = "opUser"

// OrgKey is the default context key for the organization stored in a request's
// context.
var OrgKey OrgCtxKey = "reqOrg"

// CtxReqUser returns the actor associated with this context. As it currently
// stands, this is not especially useful compared to how the actor executing the
// request is currently fetched, but it should be much more useful down the road
// with 1.0.0 and the permission system there.
func CtxReqUser(ctx context.Context) (actor.Actor, gerror.Error) {
	opUser, ok := ctx.Value(OpUserKey).(actor.Actor)
	if !ok {
		err := gerror.New("Surprisingly, there was no actor for this request, and there should have been.")
		return nil, err
	}
	return opUser, nil
}

// CtxOrg returns the organization associated with this context.
func CtxOrg(ctx context.Context) (*organization.Organization, gerror.Error) {
	org, ok := ctx.Value(OrgKey).(*organization.Organization)
	if !ok {
		err := gerror.New("No org found in this request context, but there should have been.")
		return nil, err
	}
	return org, nil
}
