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

type Policy struct {
	Name      string
	Revisions []*PolicyRevision
	org       *organization.Organization
	id        int64
}

type ByName []*Policy

func (p ByName) Len() int { return len(p) }
func (p ByName) Swap(i, j int) { p[i], p[j] = p[j], p[i] }
func (p ByName) Less(i, j int) bool { return p[i].Name < p[j].Name }

func New(org *organization.Organization, name string) (*Policy, util.Gerror) {
	var found bool
	if config.UsingDB() {
		var err error
		found, err = checkForPolicySQL(datastore.Dbh, org, name)
		if err != nil {
			gerr := util.CastErr(err)
			gerr.SetStatus(http.StatusInternalServerError)
			return nil, gerr
		}
	} else {
		ds := datastore.New()
		_, found = ds.Get(org.DataKey("policy"), name)
	}

	if found {
		err := util.Errorf("Policy '%s' already exists", name)
		err.SetStatus(http.StatusConflict)
		return nil, err
	}

	// validations?
	
	p := new(Policy)
	p.Name = name
	p.org = org

	return p, nil
}

func Get(org *organization.Organization, name string) (*Policy, util.Gerror) {
	var pol *Policy
	var found bool

	if config.UsingDB() {
		var err error
		pol, err = getPolicySQL(org, name)
		if err != nil {
			if err == sql.ErrNoRows {
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
		p, found = ds.Get(org.DataKey("policy"), name)
		if p != nil {
			pol = p.(*Policy)
			pol.org = org
		}
	}
	if !found {
		err := util.Errorf("Cannot find a policy named %s", name)
		err.SetStatus(http.StatusNotFound)
		return nil, err
	}

	return pol, nil
}

func (p *Policy) Save() util.Gerror {
	var err error
	if config.UsingDB() {
		err = p.savePolicySQL()
	} else {
		ds := datastore.New()
		ds.Set(p.org.DataKey("policy"), p.Name, p)
	}
	if err != nil {
		return util.CastErr(err)
	}
	return nil
}

func (p *Policy) Delete() util.Gerror {
	if config.UsingDB() {
		if err := p.deletePolicySQL(); err != nil {
			return util.CastErr(err)
		}
	} else {
		ds := datastore.New()
		ds.Delete(p.org.DataKey("policy"), p.Name)
	}
	return nil
}

func GetList(org *organization.Organization) ([]string, util.Gerror) {
	var polList []string
	if config.UsingDB() {
		var err error
		polList, err = getPolicyListSQL(org)
		if err != nil && !xerrors.Is(err, sql.ErrNoRows) {
			gerr := util.CastErr(err)
			return nil, gerr
		}
	} else {
		ds := datastore.New()
		polList = ds.GetList(org.DataKey("policy"))
	}

	return polList, nil
}

func (p *Policy) GetName() string {
	return p.Name
}

func (p *Policy) URLType() string {
	return "policies"
}

func (p *Policy) ContainerType() string {
	return p.URLType()
}

func (p *Policy) ContainerKind() string {
	return "containers"
}

func (p *Policy) OrgName() string {
	return p.org.Name
}

func (p *Policy) URI() string {
	return util.ObjURL(p)
}

func AllPolicies(org *organization.Organization) ([]*Policy, util.Gerror) {
	var policies []*Policy
	if config.UsingDB() {
		var err error
		policies, err = allPoliciesSQL(org)
		if err != nil && !xerrors.Is(err, sql.ErrNoRows) {
			gerr := util.CastErr(err)
			return nil, gerr
		}
	} else {
		polList, _ := GetList(org)
		policies = make([]*Policy, 0, len(polList))
		for _, p := range polList {
			po, err := Get(org, p)
			if err != nil {
				continue
			}
			policies = append(policies, po)
		}	
	}

	return policies, nil
}
