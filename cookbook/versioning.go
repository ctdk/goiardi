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

package cookbook

import (
	"fmt"
	"github.com/ctdk/goiardi/depgraph"
	"github.com/ctdk/goiardi/util"
	gversion "github.com/hashicorp/go-version"
	"strconv"
	"strings"
)

// Types, functions, and methods for dealing with cookbook versioning that
// aren't methods on the cookbooks themselves.

// VersionStrings is a type to make version strings with the format "x.y.z"
// sortable.
type VersionStrings []string

type versionConstraint gversion.Constraints

type versionConstraintError struct {
	ViolationType    int
	ParentCookbook   string
	ParentConstraint string
	ParentVersion    string
	Cookbook         string
	Constraint       string
}

func (v versionConstraint) Satisfied(head, tail *depgraph.Noun) (bool, error) {
	tMeta := tail.Meta.(*depMeta)
	var headVersion string
	var headConstraint string
	if head.Meta != nil {
		headVersion = head.Meta.(*depMeta).version
		headConstraint = head.Meta.(*depMeta).constraint.String()
	}

	verr := &versionConstraintError{ParentCookbook: head.Name, ParentVersion: headVersion, ParentConstraint: headConstraint, Cookbook: tail.Name, Constraint: v.String()}

	if tMeta.notFound {
		verr.ViolationType = CookbookNotFound
		return false, verr
	}
	if tMeta.version == "" {
		verr.ViolationType = CookbookNoVersion
		// but what constraint isn't met?
		cb, _ := Get(tMeta.organization, tail.Name)
		if cb != nil {
			badver := cb.badConstraints(v)
			verr.Constraint = strings.Join(badver, ",")
		}
		return false, verr
	}

	return true, nil
}

func (v versionConstraint) String() string {
	return gversion.Constraints(v).String()
}

func constraintPresent(constraints versionConstraint, cons string) bool {
	for _, c := range constraints {
		if c.String() == cons {
			// already in here, bail
			return true
		}
	}
	return false
}

func appendConstraint(constraints *versionConstraint, cons string) {
	if constraintPresent(*constraints, cons) {
		return
	}
	newcon, _ := gversion.NewConstraint(cons)
	*constraints = append(*constraints, newcon...)
}

func checkDependency(node *depgraph.Noun, cbName string) (*depgraph.Dependency, int, bool) {
	depName := fmt.Sprintf("%s-%s", node.Name, cbName)
	for i, d := range node.Deps {
		if depName == d.Name {
			return d, i, true
		}
	}
	return nil, -1, false
}

func splitConstraint(constraint string) (string, string, error) {
	t1 := strings.Split(constraint, " ")
	if len(t1) != 2 {
		err := fmt.Errorf("Constraint '%s' was not well-formed.", constraint)
		return "", "", err
	}
	op := t1[0]
	ver := t1[1]
	return op, ver, nil
}

func extractVerNums(cbVersion string) (maj, min, patch int64, err util.Gerror) {
	if _, err = util.ValidateAsVersion(cbVersion); err != nil {
		return 0, 0, 0, err
	}
	nums := strings.Split(cbVersion, ".")
	if len(nums) < 2 && len(nums) > 3 {
		err = util.Errorf("incorrect number of numbers in version string '%s': %d", cbVersion, len(nums))
		return 0, 0, 0, err
	}
	var vt int64
	var nerr error
	vt, nerr = strconv.ParseInt(nums[0], 0, 64)
	if nerr != nil {
		err = util.Errorf(nerr.Error())
		return 0, 0, 0, err
	}
	maj = vt
	vt, nerr = strconv.ParseInt(nums[1], 0, 64)
	if nerr != nil {
		err = util.Errorf(nerr.Error())
		return 0, 0, 0, err
	}
	min = vt
	if len(nums) == 3 {
		vt, nerr = strconv.ParseInt(nums[2], 0, 64)
		if nerr != nil {
			err = util.Errorf(nerr.Error())
			return 0, 0, 0, err
		}
		patch = vt
	} else {
		patch = 0
	}
	return maj, min, patch, nil
}

/* Version string functions to implement sorting */

func (v VersionStrings) Len() int {
	return len(v)
}

func (v VersionStrings) Swap(i, j int) {
	v[i], v[j] = v[j], v[i]
}

func (v VersionStrings) Less(i, j int) bool {
	return versionLess(v[i], v[j])
}

func versionLess(verA, verB string) bool {
	/* Chef cookbook versions are always to be in the form x.y.z (with x.y
	 * also allowed. This simplifies things a bit. */

	/* Easy comparison. False if they're equal. */
	if verA == verB {
		return false
	}

	/* Would caching the split strings ever be particularly worth it? */
	iVer := strings.Split(verA, ".")
	jVer := strings.Split(verB, ".")

	for q := 0; q < 3; q++ {
		/* If one of them doesn't actually exist, then obviously the
		 * other is bigger, and we're done. Of course this should only
		 * happen with the 3rd element. */
		if len(iVer) < q+1 {
			return true
		} else if len(jVer) < q+1 {
			return false
		}

		ic := iVer[q]
		jc := jVer[q]

		/* Otherwise, see if they're equal. If they're not, return the
		 * result of x < y. */
		ici, _ := strconv.Atoi(ic)
		jci, _ := strconv.Atoi(jc)
		if ici != jci {
			return ici < jci
		}
	}
	return false
}

/* Compares a version number against a constraint, like version 1.2.3 vs.
 * ">= 1.0.1". In this case, 1.2.3 passes. It would not satisfy "= 1.2.0" or
 * "< 1.0", though. */

func verConstraintCheck(verA, verB, op string) string {
	switch op {
	case "=":
		if verA == verB {
			return "ok"
		} else if versionLess(verA, verB) {
			/* If we want equality and verA is less than
			 * version b, since the version list is sorted
			 * in descending order we've missed our chance.
			 * So, break out. */
			return "break"
		} else {
			return "skip"
		}
	case ">":
		if verA == verB || versionLess(verA, verB) {
			return "break"
		}
		return "ok"
	case "<":
		/* return skip here because we might find what we want
		 * later. */
		if verA == verB || !versionLess(verA, verB) {
			return "skip"
		}
		return "ok"
	case ">=":
		if !versionLess(verA, verB) {
			return "ok"
		}
		return "break"
	case "<=":
		if verA == verB || versionLess(verA, verB) {
			return "ok"
		}
		return "skip"
	case "~>":
		/* only check pessimistic constraints if they can
		 * possibly be valid. */
		if versionLess(verA, verB) {
			return "break"
		}
		var upperBound string
		pv := strings.Split(verB, ".")
		if len(pv) == 3 {
			uver, _ := strconv.Atoi(pv[1])
			uver++
			upperBound = fmt.Sprintf("%s.%d", pv[0], uver)
		} else {
			uver, _ := strconv.Atoi(pv[0])
			uver++
			upperBound = fmt.Sprintf("%d.0", uver)
		}
		if !versionLess(verA, verB) && versionLess(verA, upperBound) {

			return "ok"
		}
		return "skip"
	default:
		return "invalid"
	}
}

func (v *versionConstraintError) Error() string {
	// assemble error message from what we have
	msg := fmt.Sprintf("%s: %s %s %s %s", cookbookVerErr[v.ViolationType], v.ParentCookbook, v.ParentVersion, v.Cookbook, v.Constraint)
	return msg
}

func (v *versionConstraintError) String() string {
	return v.Error()
}
