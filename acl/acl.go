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
	"github.com/ctdk/goiardi/aclhelper"
	"github.com/ctdk/goiardi/association"
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/organization"
	"github.com/ctdk/goiardi/util"
	"github.com/tideland/golib/logger"
	"os"
	"path"
)

var DefaultACLs = [5]string{
	"create",
	"read",
	"update",
	"delete",
	"grant",
}

type enforceCondition []interface{}

type Checker struct {
	org *organization.Organization
	e *casbin.SyncedEnforcer
}

// group, subkind, kind, name, perm, effect
const (
	condGroupPos = iota
	condSubkindPos
	condKindPos
	condNamePos
	condPermPos
	condEffectPos
)
const enforceEffect = "allow"

const policyFileFmt = "%s-policy.csv"

var DefaultUser = "pivotal" // should this be configurable?

func LoadACL(org *organization.Organization) error {
	m := casbin.NewModel(modelDefinition)
	if !policyExists(org, config.Config.PolicyRoot) {
		newE, err := initializeACL(org, m)
		if err != nil {
			return err
		}
		c := &Checker{org: org, e: newE}
		org.PermCheck = c
		return nil
	}
	pa, err := loadPolicyAdapter(org)
	if err != nil {
		return err
	}
	e := casbin.NewSyncedEnforcer(m, pa, true)
	c := &Checker{org: org, e: e}
	org.PermCheck = c

	return nil
}

func initializeACL(org *organization.Organization, m model.Model) (*casbin.SyncedEnforcer, error) {
	if err := initializePolicy(org, config.Config.PolicyRoot); err != nil {
		return nil, err
	}
	adp, err := loadPolicyAdapter(org)
	if err != nil {
		return nil, err
	}
	e := casbin.NewSyncedEnforcer(m, adp, true)
	
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
	if _, err = p.WriteString(defaultPolicySkel); err != nil {
		return  err
	}
	return nil
}

func (c *Checker) CheckItemPerm(item aclhelper.Item, doer aclhelper.Actor, perm string) (bool, util.Gerror) {
	specific := buildEnforcingSlice(item, doer, perm)
	var chkSucceeded bool

	// try the specific check first, then the general
	if chkSucceeded = c.e.Enforce(specific...); !chkSucceeded {
		chkSucceeded = c.e.Enforce(specific.general()...)
	}
	if chkSucceeded {
		return true, nil
	}

	// check out failure conditions
	if !c.isPermValid(item, perm) {
		err := util.Errorf("invalid perm %s for %s-%s", perm, item.ContainerKind(), item.ContainerType())
		return false, err
	}

	_, err := association.TestAssociation(doer, c.org)
	if err != nil {
		return false, err
	}

	return false, nil
}

func (c *Checker) RootCheckPerm(doer aclhelper.Actor, perm string) (bool, util.Gerror) {
	return c.CheckItemPerm(c.org, doer, perm)
}

func buildEnforcingSlice(item aclhelper.Item, doer aclhelper.Actor, perm string) enforceCondition {
	cond := []interface{}{doer.GetName(), item.ContainerType(), item.ContainerKind(), item.GetName(), perm, enforceEffect}
	return enforceCondition(cond)
}

func (e enforceCondition) general() enforceCondition {
	g := make([]interface{}, len(e))
	for i, v := range e {
		g[i] = v
	}
	g[condNamePos] = "default"
	return enforceCondition(g)
}

func (c *Checker) isPermValid (item aclhelper.Item, perm string) bool {
	// pare down the list to check a little
	fPass := c.e.GetFilteredPolicy(condSubkindPos, item.ContainerType())
	validPerms := make(map[string]bool)
	for _, p := range fPass {
		if p[condKindPos] == item.ContainerKind() {
			validPerms[p[condPermPos]] = true
		}
	}
	return validPerms[perm]
}

// TODO: Determine what's actually needed with these...? There might not be much
// for this.
func (c *Checker) AddACLRole(gRole aclhelper.Member) error {

	return nil
}

func (c *Checker) RemoveACLRole(gRole aclhelper.Member) error {
	c.e.DeleteRole(gRole.ACLName())
	return nil
}

func (c *Checker) AddMembers(gRole aclhelper.Member, adding []aclhelper.Member) error {
	for _, m := range adding {
		c.e.AddRoleForUser(m.ACLName(), gRole.ACLName())
	}
	logger.Debugf("added %d members to %s ACL role", len(adding), gRole.GetName())

	return nil
}

func (c *Checker) RemoveMembers(gRole aclhelper.Member, removing []aclhelper.Member) error {
	for _, m := range removing {
		c.e.DeleteRoleForUser(m.ACLName(), gRole.ACLName())
	}
	logger.Debugf("deleted %d members from %s ACL role", len(removing), gRole.GetName())

	return nil
}

func (c *Checker) Enforcer() *casbin.SyncedEnforcer {
	return c.e
}
