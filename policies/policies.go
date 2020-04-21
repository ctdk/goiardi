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
	"github.com/ctdk/goiardi/organization"
	"github.com/ctdk/goiardi/util"
)

type Policy struct {
	Name      string
	URI       string
	Revisions []*PolicyRevision
	org       *organization.Organization
}

// PolicyRevision currently contains more map[string]interface{} fields than I
// feel all that comfortable with. At the moment, however, it kind of looks like
// the structure of some of these fields is a bit freeform and inconsistent. For
// the time being, we'll stick with this and change them to be real types when
// possible down the road.
type PolicyRevision struct {
	RevisionId string
	Name string
	RunList []string
	CookbookLocks map[string]interface{}
	Default map[string]interface{}
	Override map[string]interface{}
	SolutionDependencies map[string]interface{}
	org *organization.Organization
}

func New(name string, uri string, polOrg *organization.Organization) (*Policy, util.Gerror) {
	p := new(Policy)
	p.Name = name
	p.URI = uri
	p.org = polOrg

	// Don't forget to save!
	if err := p.Save(); err != nil {
		return nil, err
	}

	return p, nil
}

func Get() error {
	return nil
}

func (p *Policy) Save() util.Gerror {

	return nil
}
