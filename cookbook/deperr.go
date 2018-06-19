/*
 * Copyright (c) 2013-2017, Jeremy Bingham (<jeremy@goiardi.gl>)
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

package cookbook

import (
	"fmt"
	"github.com/ctdk/goiardi/depgraph"
	"github.com/ctdk/goiardi/organization"
	"github.com/ctdk/goiardi/util"
	"github.com/tideland/golib/logger"
	"sort"
	"strings"
)

// Handling dependency errors, woo.

const (
	CookbookNotFound int = iota
	CookbookNoVersion
)

var cookbookVerErr = map[int]string{CookbookNotFound: "not found", CookbookNoVersion: "no version"}

type depMeta struct {
	version    string
	constraint versionConstraint
	notFound   bool
	noVersion  bool
	organization *organization.Organization
}

type DependsError struct {
	depErr *depgraph.ConstraintError
	organization *organization.Organization
}

func (d *DependsError) Error() string {
	errMap := d.ErrMap()
	return errMap["message"].(string)
}

func (d *DependsError) String() string {
	return d.Error()
}

func (d *DependsError) ErrMap() map[string]interface{} {
	errMap := make(map[string]interface{})

	allMsgs := make([]string, 0)
	notFound := make([]*versionConstraintError, 0)
	mostConstrained := make([]*versionConstraintError, 0)
	noVersion := make([]*versionConstraintError, 0)
	unsatisfiable := make([]*versionConstraintError, 0)

	for _, ce := range d.depErr.Violations {
		logger.Debugf("dependency violation: %+v", ce)
		verr := ce.Err.(*versionConstraintError)
		var unsat bool
		if verr.ParentCookbook != "^runlist_root^" {
			unsat = true
			unsatisfiable = append(unsatisfiable, verr)
		}
		if verr.ViolationType == CookbookNotFound {
			notFound = append(notFound, verr)
		} else {
			if unsat {
				mostConstrained = append(mostConstrained, verr)

			} else {
				noVersion = append(noVersion, verr)
			}
		}
	}
	notFoundStr := make([]string, 0)
	mostConstrainedStr := make([]string, 0)
	noVersionStr := make([]string, 0)
	unsatisfiableStr := make([]string, 0)

	if len(notFound) > 0 {
		for _, v := range notFound {
			notFoundStr = append(notFoundStr, v.Cookbook)
		}
		sort.Strings(notFoundStr)
		notFoundStr = util.RemoveDupStrings(notFoundStr)
	}
	if len(noVersion) > 0 {
		for _, v := range noVersion {
			noVersionStr = append(noVersionStr, fmt.Sprintf("(%s %s)", v.Cookbook, v.Constraint))
		}
		sort.Strings(noVersionStr)
		noVersionStr = util.RemoveDupStrings(noVersionStr)
	}
	if len(mostConstrained) > 0 {
		for _, v := range mostConstrained {
			failedDep := fmt.Sprintf("%s %s -> [%s %s]", v.Cookbook, v.Constraint, v.ParentCookbook, v.ParentConstraint)
			mostConstrainedStr = append(mostConstrainedStr, failedDep)
		}
		sort.Strings(mostConstrainedStr)
		mostConstrainedStr = util.RemoveDupStrings(mostConstrainedStr)
	}

	// Craft the messages
	if len(unsatisfiable) > 0 {
		for _, v := range unsatisfiable {
			var doesntExist string
			if v.ViolationType == CookbookNotFound {
				doesntExist = ", which does not exist,"
			}
			msgTmp := fmt.Sprintf("Unable to satisfy constraints on package %s%s due to solution constraint (%s %s). Solution constraints that may result in a constraint on %s: [(%s = %s) -> (%s %s)]", v.Cookbook, doesntExist, v.ParentCookbook, v.ParentConstraint, v.Cookbook, v.ParentCookbook, v.ParentVersion, v.Cookbook, v.Constraint)
			allMsgs = append(allMsgs, msgTmp)
			unsatisfiableStr = append(unsatisfiableStr, fmt.Sprintf("(%s %s)", v.ParentCookbook, v.ParentConstraint))
		}
		sort.Strings(unsatisfiableStr)
		util.RemoveDupStrings(unsatisfiableStr)
	}

	if len(notFoundStr) > 0 || len(noVersionStr) > 0 {
		msgTmp := "Run list contains invalid items:"
		if len(notFoundStr) > 0 {
			var werd string
			if len(notFoundStr) == 1 {
				werd = "cookbook"
			} else {
				werd = "cookbooks"
			}
			msgTmp = fmt.Sprintf("%s no such %s %s", msgTmp, werd, strings.Join(notFoundStr, ", "))
		}
		if len(noVersionStr) > 0 {
			msgTmp = fmt.Sprintf("%s no versions match the constraints on cookbook %s", msgTmp, strings.Join(noVersionStr, ", "))
		}
		msgTmp = strings.Join([]string{msgTmp, "."}, "")
		allMsgs = append(allMsgs, msgTmp)
	}

	errMap["message"] = strings.Join(allMsgs, "\n")

	errMap["non_existent_cookbooks"] = notFoundStr
	if len(unsatisfiable) > 0 {
		errMap["unsatisfiable_run_list_item"] = strings.Join(unsatisfiableStr, ", ")
		errMap["most_constrained_cookbooks"] = mostConstrainedStr
	} else {
		errMap["cookbooks_with_no_versions"] = noVersionStr
	}

	return errMap
}
