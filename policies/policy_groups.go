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
	"database/sql"
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/datastore"
	"github.com/ctdk/goiardi/organization"
	"github.com/ctdk/goiardi/util"
	"golang.org/x/xerrors"
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
	PolicyInfo map[string]*pgRevisionInfo
	org        *organization.Organization
	id         int64
}

func NewPolicyGroup(org *organization.Organization, name string) (*PolicyGroup, util.Gerror) {
	var found bool
	if config.UsingDB() {
		var err error
		found, err = checkForPolicyGroupSQL(datastore.Dbh, org, name)
		if err != nil {
			gerr := util.CastErr(err)
			gerr.SetStatus(http.StatusInternalServerError)
			return nil, gerr
		}
	} else {
		ds := datastore.New()
		_, found = ds.Get(org.DataKey("policy_group"), name)
	}

	if found {
		err := util.Errorf("Policy group '%s' already exists", name)
		err.SetStatus(http.StatusConflict)
		return nil, err
	}

	m := make(map[string]*pgRevisionInfo)
	pg := &PolicyGroup{Name: name, PolicyInfo: m, org: org}
	return pg, nil
}

func GetPolicyGroup(org *organization.Organization, name string) (*PolicyGroup, util.Gerror) {
	var pg *PolicyGroup
	var found bool

	if config.UsingDB() {
		var err error
		pg, err = getPolicyGroupSQL(org, name)
		if err != nil {
			if xerrors.Is(err, sql.ErrNoRows) {
				found = false
			} else {
				gerr := util.CastErr(err)
				gerr.SetStatus(http.StatusInternalServerError)
				return nil, gerr
			}
		} else {
			found = true
		}
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
		return nil, err
	}

	return pg, nil
}

func (pg *PolicyGroup) Save() util.Gerror {
	var err error
	if config.UsingDB() {
		err = pg.savePolicyGroupSQL()
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
		if err := pg.deletePolicyGroupSQL(); err != nil {
			return util.CastErr(err)
		}
	} else {
		ds := datastore.New()
		ds.Delete(pg.org.DataKey("policy_group"), pg.Name)
	}

	return nil
}

func (pg *PolicyGroup) AddPolicy(pr *PolicyRevision) util.Gerror {
	if config.UsingDB() {
		if err := pg.addPolicySQL(pr); err != nil {
			gerr := util.CastErr(err)
			gerr.SetStatus(http.StatusInternalServerError)
			return gerr
		}
	}

	pi := &pgRevisionInfo{PolicyName: pr.PolicyName(), RevisionId: pr.RevisionId}
	pg.PolicyInfo[pr.PolicyName()] = pi
	return nil
}

func (pg *PolicyGroup) RemovePolicy(policyName string) util.Gerror {
	if config.UsingDB() {
		if err := pg.removePolicySQL(policyName); err != nil && !xerrors.Is(err, sql.ErrNoRows) {
			gerr := util.CastErr(err)
			gerr.SetStatus(http.StatusInternalServerError)
			return gerr
		}
	}

	delete(pg.PolicyInfo, policyName)

	return nil
}

func (pg *PolicyGroup) NumPolicies() int {
	return len(pg.PolicyInfo)
}

// this is mostly for in-mem when deleting specific policy revisions without
// deleting the whole policy.
func (pg *PolicyGroup) removePolicyByRevision(policyName string, revisionId string) util.Gerror {
	if config.UsingDB() {
		return util.Errorf("removePolicyByRevision is only useful in in-memory mode when deleting just a specific policy revision and not the whole policy.")
	}

	pi, ok := pg.PolicyInfo[policyName]
	if !ok {
		return util.Errorf("policy %s not found in policy group %s", policyName, pg.Name)
	}
	if revisionId != pi.RevisionId {
		return util.Errorf("policy group %s does not contain revision '%s' of policy %s (but does contain '%s')", pg.Name, revisionId, policyName, pi.RevisionId)
	}
	return pg.RemovePolicy(policyName)
}

func (pg *PolicyGroup) GetPolicy(name string) (*PolicyRevision, util.Gerror) {
	pi, ok := pg.PolicyInfo[name]
	if !ok {
		return nil, util.Errorf("Policy %s not associated with policy group %s", name, pg.Name)
	}

	p, err := Get(pg.org, pi.PolicyName)
	if err != nil {
		return nil, err
	}

	pr, err := p.GetPolicyRevision(pi.RevisionId)
	if err != nil {
		return nil, err
	}

	return pr, nil
}

// Ooof, that's an icky return type
func (pg *PolicyGroup) GetPolicyMap() map[string]map[string]string {
	pm := make(map[string]map[string]string, len(pg.PolicyInfo))
	for k, v := range pg.PolicyInfo {
		m := make(map[string]string, 1)
		m["revision_id"] = v.RevisionId
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

func GetAllPolicyGroups(org *organization.Organization) ([]*PolicyGroup, util.Gerror) {
	if config.UsingDB() {
		allPgs, err := getAllPolicyGroupsSQL(org)
		if err != nil {
			var s int
			if xerrors.Is(err, sql.ErrNoRows) {
				s = http.StatusNotFound
			} else {
				s = http.StatusInternalServerError
			}
			gerr := util.CastErr(err)
			gerr.SetStatus(s)
			return nil, gerr
		}
		return allPgs, nil
	}

	return nil, nil
}
