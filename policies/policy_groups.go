/*
 * Copyright (c) 2013-2020, Jeremy Bingham (<jeremy@goiardi.gl>)
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

/*
 * At last, goiardi's going to get policyfile support.
 */

package policies

import (
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/datastore"
	"github.com/ctdk/goiardi/organization"
	"github.com/ctdk/goiardi/util"
	"net/http"
)

// It may be better to only use this for policy_group -> policy/policy rev links
// now that I've been thinking about it.
type pgRevisionInfo struct {
	PolicyId    int64  `json:"policy_id"`
	PolicyRevId int64  `json:"policy_rev_id"`
	PolicyName  string `json:"name"`
	RevisionId  string `json:"revision_id"`
}

type PolicyGroup struct {
	Name       string
	Policies   map[string]*PolicyRevision // NB: only for in-mem
	policyInfo map[string]*pgRevisionInfo // NB: for SQL
	org        *organization.Organization
	id         int64
}

func (pgr *pgRevisionInfo) getRevId() string {
	return pgr.RevisionId
}

type revisionator interface {
	getRevId() string
}

func NewPolicyGroup(org *organization.Organization, name string) (*PolicyGroup, util.Gerror) {
	// check for existing, validate name, yadda yadda
	pg := &PolicyGroup{Name: name, org: org}
	return pg, nil
}

func GetPolicyGroup(org *organization.Organization, name string) (*PolicyGroup, util.Gerror) {
	var pg *PolicyGroup
	var found bool

	if config.UsingDB() {

	} else {
		ds := datastore.New()
		var p interface{}
		p, found = ds.Get(org.DataKey("policy_group"), name)
		if p != nil {
			pg = p.(*PolicyGroup)
			pg.org = org
		}
	}
	if !found {
		err := util.Errorf("Cannot find a policy group named %s", name)
		err.SetStatus(http.StatusNotFound)
	}

	return pg, nil
}

func (pg *PolicyGroup) Save() util.Gerror {
	var err error
	if config.UsingDB() {

	} else {
		ds := datastore.New()
		ds.Set(pg.org.DataKey("policy_group"), pg.Name, pg)
	}
	if err != nil {
		return util.CastErr(err)
	}

	return nil
}

func (pg *PolicyGroup) Delete() util.Gerror {
	if config.UsingDB() {

	} else {
		ds := datastore.New()
		ds.Delete(pg.org.DataKey("policy_group"), pg.Name)
	}

	return nil
}

func (pg *PolicyGroup) AddPolicy(pr *PolicyRevision) util.Gerror {
	if config.UsingDB() {

	}

	pg.Policies[pr.PolicyName()] = pr
	return nil
}

func (pg *PolicyGroup) RemovePolicy(policyName string) util.Gerror {
	if config.UsingDB() {

	}
	delete(pg.Policies, policyName)

	return nil
}

// Ooof, that's an icky return type
func (pg *PolicyGroup) GetPolicyMap() map[string]map[string]string {
	var revsies map[string]revisionator

	if config.UsingDB() {
		for k, v := range pg.policyInfo {
			revsies[k] = v
		}
	} else {
		for k, v := range pg.Policies {
			revsies[k] = v
		}
	}

	pm := make(map[string]map[string]string, len(revsies))

	for k, v := range revsies {
		m := make(map[string]string, 1)
		m["revision_id"] = v.getRevId()
		pm[k] = m
	}

	return pm
}

func (pg *PolicyGroup) GetName() string {
	return pg.Name
}

func (pg *PolicyGroup) URLType() string {
	return "policy_groups"
}

func (pg *PolicyGroup) ContainerType() string {
	return pg.URLType()
}

func (pg *PolicyGroup) ContainerKind() string {
	return "containers"
}

func (pg *PolicyGroup) OrgName() string {
	return pg.org.Name
}

func (pg *PolicyGroup) URI() string {
	return util.ObjURL(pg)
}
