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
	"sort"
	"strings"
	"time"
)

// PolicyRevision currently contains more map[string]interface{} fields than I
// feel all that comfortable with. At the moment, however, it kind of looks like
// the structure of some of these fields is a bit freeform and inconsistent. For
// the time being, we'll stick with this and change them to be real types when
// possible down the road.
type PolicyRevision struct {
	RevisionId           string                 `json:"revision_id"`
	RunList              []string               `json:"run_list"`
	CookbookLocks        map[string]interface{} `json:"cookbook_locks"`
	Default              map[string]interface{} `json:"default_attributes"`
	Override             map[string]interface{} `json:"override_attributes"`
	SolutionDependencies map[string]interface{} `json:"solution_dependencies"`
	creationTime         time.Time
	pol                  *Policy
	id                   int64
}

// Types to help sorting output

type ByRevTime []*PolicyRevision

func (pr ByRevTime) Len() int           { return len(pr) }
func (pr ByRevTime) Swap(i, j int)      { pr[i], pr[j] = pr[j], pr[i] }
func (pr ByRevTime) Less(i, j int) bool { return pr[i].creationTime.Before(pr[j].creationTime) }

type ByRevId []*PolicyRevision

func (pr ByRevId) Len() int           { return len(pr) }
func (pr ByRevId) Swap(i, j int)      { pr[i], pr[j] = pr[j], pr[i] }
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
		_, found = p.findRevisionId(revisionId)
	}

	if found {
		err := util.Errorf("Policy revision '%s' for policy '%s' already exists", revisionId, p.Name)
		err.SetStatus(http.StatusConflict)
		return nil, err
	}

	rev := &PolicyRevision{
		RevisionId: revisionId,
		pol:        p,
	}

	return rev, nil
}

func (p *Policy) NewPolicyRevisionFromJSON(policyRevJSON map[string]interface{}) (*PolicyRevision, util.Gerror) {
	revId, ok := policyRevJSON["revision_id"].(string)
	if !ok {
		err := util.Errorf("Invalid or missing 'revision_id' field. Contents were: '%+v'", policyRevJSON["revision_id"])
		return nil, err
	}

	pr, err := p.NewPolicyRevision(revId)
	if err != nil {
		return nil, err
	}

	if policyRevJSON["run_list"], err = util.ValidateRunList(policyRevJSON["run_list"]); err != nil {
		return nil, err
	}
	mapAttrs := []string{"cookbook_locks", "default_attributes", "override_attributes", "solution_dependencies"}
	for _, a := range mapAttrs {
		if policyRevJSON[a], err = util.ValidateAttributes(a, policyRevJSON[a]); err != nil {
			return nil, err
		}
	}

	// theoretically all should be well
	pr.RunList = policyRevJSON["run_list"].([]string)
	pr.CookbookLocks = policyRevJSON["cookbook_locks"].(map[string]interface{})
	pr.Default = policyRevJSON["default_attributes"].(map[string]interface{})
	pr.Override = policyRevJSON["override_attributes"].(map[string]interface{})
	pr.SolutionDependencies = policyRevJSON["solution_dependencies"].(map[string]interface{})

	return pr, nil
}

func (p *Policy) findRevisionId(revisionId string) (int, bool) {
	pRevLen := len(p.Revisions)
	sort.Sort(ByRevId(p.Revisions))
	i := sort.Search(pRevLen, func(i int) bool { return p.Revisions[i].RevisionId >= revisionId })
	return i, i < pRevLen && p.Revisions[i].RevisionId == revisionId
}

func (p *Policy) GetPolicyRevision(revisionId string) (*PolicyRevision, util.Gerror) {
	// Shouldn't need to re-fetch from the db. Famous last words, of course,
	// but we could fall back to the db if it's not available to this
	// policy in its in-memory slice of revisions.

	// set up a handy error ahead of time since there's a few places it can
	// be returned.

	prErr := util.Errorf("no revisions found for policy '%s'", p.Name)
	prErr.SetStatus(http.StatusNotFound)

	if len(p.Revisions) == 0 {
		return nil, prErr
	}

	i, found := p.findRevisionId(revisionId)
	if !found {
		return nil, prErr
	}

	return p.Revisions[i], nil
}

func (p *Policy) MostRecentRevision() (*PolicyRevision, util.Gerror) {
	// see the note above
	prErr := util.Errorf("no revisions found for policy '%s'", p.Name)
	prErr.SetStatus(http.StatusNotFound)

	if len(p.Revisions) == 0 {
		return nil, prErr
	}

	sort.Sort(sort.Reverse(ByRevTime(p.Revisions)))

	return p.Revisions[0], nil
}

func (pr *PolicyRevision) Save() util.Gerror {
	if config.UsingDB() {
		if err := pr.saveRevisionSQL(); err != nil {
			return util.CastErr(err)
		}
		return nil
	}

	pr.creationTime = time.Now()

	// TODO: insert this rather than merely appending
	_, found := pr.pol.findRevisionId(pr.RevisionId)

	if found {
		err := util.Errorf("policy '%s' already has revision '%s'", pr.pol.Name, pr.RevisionId)
		err.SetStatus(http.StatusConflict)
		return err
	}

	pr.pol.Revisions = append(pr.pol.Revisions, pr)
	return nil
}

func (pr *PolicyRevision) Delete() util.Gerror {
	if config.UsingDB() {
		if err := pr.deleteRevisionSQL(); err != nil {
			return util.CastErr(err)
		}
		// not returning here so we can remove the pr from the parent
		// policy in memory as well
	}
	i, _ := pr.pol.findRevisionId(pr.RevisionId)
	pr.pol.Revisions = append(pr.pol.Revisions[:i], prl.pol.Revisions[i+1:]...)

	return nil
}

func (pr *PolicyRevision) getRevId() string {
	return pr.RevisionId
}

func (pr *PolicyRevision) PolicyName() string {
	return pr.pol.Name
}

func (pr *PolicyRevision) GetName() string {
	return strings.Join([]string{pr.pol.Name, pr.RevisionId}, "%%")
}

func (pr *PolicyRevision) URLType() string {
	return pr.pol.URLType()
}

func (pr *PolicyRevision) ContainerType() string {
	return pr.pol.ContainerType()
}

func (pr *PolicyRevision) ContainerKind() string {
	return pr.pol.ContainerKind()
}

func (pr *PolicyRevision) OrgName() string {
	return pr.pol.org.Name
}

func (pr *PolicyRevision) URI() string {
	return util.CustomObjURL(pr.pol, strings.Join([]string{"revisions", pr.RevisionId}, "/"))
}
