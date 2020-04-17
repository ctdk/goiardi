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
	Revisions []string // We may want this to be a new type
	org       *organization.Organization
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
