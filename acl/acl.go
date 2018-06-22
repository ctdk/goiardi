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
	"fmt"
	"github.com/casbin/casbin"
	"github.com/casbin/casbin/model"
	"github.com/casbin/casbin/persist"
	"github.com/casbin/casbin/persist/file-adapter"
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/organization"
	"os"
	"path"
	"strings"
)

var DefaultACLs = [5]string{
	"create",
	"read",
	"update",
	"delete",
	"grant",
}

type ACLOwner interface {
	GetName() string
	ContainerKind() string
	ContainerType() string
}

const policyFileFmt = "%s-policy.csv"

var DefaultUser = "pivotal" // should this be configurable?

func loadACL(org *organization.Organization) (*casbin.Enforcer, error) {
	m := casbin.NewModel(strings.TrimSpace(modelDefinition))
	if !policyExists(org, config.Config.PolicyRoot) {
		return initializeACL(org, m)
	}
	pa, err := loadPolicyAdapter(org)
	_ = pa
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func initializeACL(org *organization.Organization, m model.Model) (*casbin.Enforcer, error) {
	if err := initializePolicy(org, config.Config.PolicyRoot); err != nil {
		return nil, err
	}
	adp, err := loadPolicyAdapter(org)
	if err != nil {
		return nil, err
	}
	e := casbin.NewEnforcer(m, adp)
	
	return e, nil
}

// TODO: When 1.0.0-dev starts wiring in the DBs, set up DB adapters for 
// policies. Until that time, set up a file backed one.
func loadPolicyAdapter(org *organization.Organization) (persist.Adapter, error) {
	if config.UsingDB() {

	}
	return loadPolicyFileAdapter(org, config.Config.PolicyRoot)
}

func loadPolicyFileAdapter(org *organization.Organization, policyRoot string) (persist.Adapter, error) {
	if !policyExists(org, policyRoot) {
		err := fmt.Errorf("Cannot load ACL policy for organization %s: file already exists.", org.Name)
		return nil, err
	}

	policyPath := makePolicyPath(org, policyRoot)
	adp := fileadapter.NewAdapter(policyPath)
	return adp, nil
}

func makePolicyPath(org *organization.Organization, policyRoot string) string {
	fn := fmt.Sprintf(policyFileFmt, org.Name)
	policyPath := path.Join(policyRoot, fn)
	return policyPath
}

// TODO: don't pass in policyRoot -- it won't be too relevant with the DB
// versions
func policyExists(org *organization.Organization, policyRoot string) bool {
	policyPath := makePolicyPath(org, policyRoot)
	_, err := os.Stat(policyPath)
	return !os.IsNotExist(err)
}

func initializePolicy(org *organization.Organization, policyRoot string) error {
	if policyExists(org, policyRoot) {
		perr := fmt.Errorf("ACL policy for organization %s already exists, cannot initialize!", org.Name)
		return perr
	}

	policyPath := makePolicyPath(org, policyRoot)
	p, err := os.OpenFile(policyPath, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	defer p.Close()
	if _, err = p.WriteString(strings.TrimSpace(defaultPolicySkel)); err != nil {
		return  err
	}
	return nil
}
