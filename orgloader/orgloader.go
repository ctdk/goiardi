/*
 * Copyright (c) 2013-2018, Jeremy Bingham (<jeremy@goiardi.gl>)
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

// Package orgloader is a wrapper around the organization and acl packages to
// make loading the ACL object into the organization easier.
package orgloader

import (
	"github.com/ctdk/goiardi/acl"
	"github.com/ctdk/goiardi/organization"
	"github.com/ctdk/goiardi/util"
)

func Get(name string) (*organization.Organization, util.Gerror) {
	org, err := organization.Get(name)
	if err != nil {
		return nil, err
	}
	aclErr := acl.LoadACL(org)
	if aclErr != nil {
		return nil, util.CastErr(aclErr)
	}
	return org, nil
}

func New(name, fullName string) (*organization.Organization, util.Gerror) {
	org, err := organization.New(name, fullName)
	if err != nil {
		return nil, err
	}
	aclErr := acl.LoadACL(org)
	if aclErr != nil {
		return nil, util.CastErr(aclErr)
	}
	return org, nil
}

func AllOrganizations() ([]*organization.Organization, error) {
	orgs, err := organization.AllOrganizations()
	if err != nil {
		return nil, err
	}
	for _, o := range orgs {
		aerr := acl.LoadACL(o)
		if aerr != nil {
			return nil, aerr
		}
	}
	return orgs, nil
}
