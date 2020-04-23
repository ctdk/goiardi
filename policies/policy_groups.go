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

type PolicyGroup struct {
	Name string
	Policies map[string]*PolicyRevision
	org *organization.Organization
	id int64
}

func NewPolicyGroup() (*PolicyGroup, util.Gerror) {
	return nil, nil
}

func GetPolicyGroup() (*PolicyGroup, util.Gerror) {
	return nil, nil
}

func (pg *PolicyGroup) Save() util.Gerror {

	return nil
}

func (pg *PolicyGroup) AddPolicy() util.Gerror {

	return nil
}

func (pg *PolicyGroup) RemovePolicy() util.Gerror {

	return nil
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
