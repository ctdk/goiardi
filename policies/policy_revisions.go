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
	"github.com/ctdk/goiardi/util"
	"net/http"
	"strings"
	"time"
)

// PolicyRevision currently contains more map[string]interface{} fields than I
// feel all that comfortable with. At the moment, however, it kind of looks like
// the structure of some of these fields is a bit freeform and inconsistent. For
// the time being, we'll stick with this and change them to be real types when
// possible down the road.
type PolicyRevision struct {
	RevisionId string
	RunList []string
	CookbookLocks map[string]interface{}
	Default map[string]interface{}
	Override map[string]interface{}
	SolutionDependencies map[string]interface{}
	creationTime time.Time
	pol *Policy
	id int64
}

// Types to help sorting output

type ByRevTime []*PolicyRevision

func (pr ByRevTime) Len() int { return len(pr) }
func (pr ByRevTime) Swap(i, j int) { pr[i], pr[j] = pr[j], pr[i] }
func (pr ByRevTime) Less(i, j int) bool { return pr[i].creationTime.Before(pr[j].creationTime) }

type ByRevId []*PolicyRevision

func (pr ByRevId) Len() int { return len(pr) }
func (pr ByRevId) Swap(i, j int) { pr[i], pr[j] = pr[j], pr[i] }
func (pr ByRevId) Less(i, j int) bool { return pr[i].RevisionId < pr[j].RevisionId }


// These methods are attached to Policy, not standalone functions.

func (p *Policy) NewPolicyRevision(revisionId string) (*PolicyRevision, util.Gerror) {
	var found bool
	if config.UsingDB() {
		var err error
		found, err = p.checkForRevisionSQL(datastore.Dbh, revisionId)
		if err != nil {
			gerr := util.CastErr(err)
			gerr.SetStatus(http.StatusInternalServerError)
			return nil, gerr
		}
	} else {
		// hrmph. Brute force it.
		for _, v := range p.Revisions {
			if v.RevisionId == revisionId {
				found = true
				break
			}
		}
	}

	if found {
		err := util.Errorf("Policy revision '%s' for policy '%s' already exists", revisionId, p.Name)
		err.SetStatus(http.StatusConflict)
		return nil, err
	}

	rev := &PolicyRevision{
		RevisionId: revisionId,
		pol: p,
	}

	return rev, nil
}

func (p *Policy) NewPolicyRevisionFromJSON(policyRevJSON map[string]interface{}) (*PolicyRevision, util.Gerror) {
	_ = policyRevJSON

	return nil, nil
}

func (p *Policy) GetPolicyRevision() (*PolicyRevision, util.Gerror) {

	return nil, nil
}

func (p *Policy) MostRecentRevision() *PolicyRevision {

	return nil
}

func (pr *PolicyRevision) Save() util.Gerror {

	return nil
}

func (pr *PolicyRevision) PolicyName() string {
	return pr.pol.Name
}

func (pr *PolicyRevision) GetName() string {
	return strings.Join([]string{pr.pol.Name, pr.RevisionId}, "%%")
}

func (pr *PolicyRevision) URLType() string {
	return "policies"
}

func (pr *PolicyRevision) ContainerType() string {
	return p.URLType()
}

func (pr *PolicyRevision) ContainerKind() string {
	return "containers"
}

func (pr *PolicyRevision) OrgName() string {
	return pr.pol.org.Name
}

func (pr *PolicyRevision) URI() string {
	return util.CustomObjURL(pr.pol, strings.Join([]string{"revisions", pr.RevisionId}, "/"))
}
