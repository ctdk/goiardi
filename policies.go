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

package main

import (
	"encoding/json"
	"github.com/ctdk/goiardi/actor"
	"github.com/ctdk/goiardi/orgloader"
	"github.com/ctdk/goiardi/policy"
	"github.com/ctdk/goiardi/reqctx"
	"github.com/ctdk/goiardi/util"
	"github.com/gorilla/mux"
	"net/http"
)

const (
	policyListLen = 1
	policyDetailLen = 2
)

func policyHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	org, orgerr := orgloader.Get(vars["org"])
	if orgerr != nil {
		jsonErrorReport(w, r, orgerr.Error(), orgerr.Status())
		return
	}

	opUser, oerr := reqctx.CtxReqUser(r.Context())
	if oerr != nil {
		jsonErrorReport(w, r, oerr.Error(), oerr.Status())
		return
	}

	pathArray := splitPath(r.URL.Path)[2:]
	policyName := vars["name"]

	if f, ferr := org.PermCheck.CheckContainerPerm(opUser, "policies", "read"); ferr != nil {
		jsonErrorReport(w, r, ferr.Error(), ferr.Status())
		return
	} else if !f {
		jsonErrorReport(w, r, "You do not have permission to do that", http.StatusForbidden)
		return
	}

	// HEAD check
	if r.Method == http.MethodHead {
		permCheck := func(r *http.Request, policyName string, opUser actor.Actor) util.Gerror {
			if f, ferr := org.PermCheck.CheckContainerPerm(opUser, "policies", "read"); ferr != nil {
				return ferr
			}
			if !f {
				return headForbidden()
			}
			return nil
		}
		headChecking(w, r, opUser, org, policyName, policy.DoesPolicyExist, permCheck)
		return
	}

	// knock out the definitely disallowed methods
	if r.Method != http.MethodGet && r.Method != http.MethodDelete {
		jsonErrorReport(w, r, "Unrecognized method!", http.StatusMethodNotAllowed)
		return
	}

	// is this truly the best way?
	policyResponse := make(map[string]map[string]interface{})

	switch len(pathArray) {
	case policyListLen:
		// the list only allows GET
		if r.Method != http.MethodGet {
			jsonErrorReport(w, r, "Unrecognized method!", http.StatusMethodNotAllowed)
			return
		}
		allPol, err := policy.AllPolicies(org)
		if err != nil {
			jsonErrorReport(w, r, err.Error(), err.Status())
			return
		}
		for _, p := range allPol {
			polR := make(map[string]interface{})
			polR["uri"] = p.URI()
			polR["revisions"] = p.RevisionResponse()
			policyResponse[p.Name] = polR
		}
	case policyDetailLen:
		p, err := policy.Get(org, policyName)
		if err != nil {
			jsonErrorReport(w, r, err.Error(), err.Status())
			return
		}
		policyResponse["revisions"] = p.RevisionResponse()

		// If we're deleting, check that perm and delete accordingly.
		if r.Method == http.MethodDelete {
			if f, ferr := org.PermCheck.CheckContainerPerm(opUser, "policies", "read"); ferr != nil {
				jsonErrorReport(w, r, ferr.Error(), ferr.Status())
				return
			} else if !f {
				jsonErrorReport(w, r, "You do not have permission to do that", http.StatusForbidden)
				return
			}
			err = p.Delete()
			if err != nil {
				jsonErrorReport(w, r, err.Error(), err.Status())
				return
			}
		}
	default:
		jsonErrorReport(w, r, "Not sure how we got here, but this URL has an improper number of elements.", http.StatusBadRequest)
		return
	}

	enc := json.NewEncoder(w)
	if err := enc.Encode(&policyResponse); err != nil {
		jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
	}
}

/*
func policyRevisionHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	org, orgerr := orgloader.Get(vars["org"])
	if orgerr != nil {
		jsonErrorReport(w, r, orgerr.Error(), orgerr.Status())
		return
	}

	opUser, oerr := reqctx.CtxReqUser(r.Context())
	if oerr != nil {
		jsonErrorReport(w, r, oerr.Error(), oerr.Status())
		return
	}
}
*/

/*
func policyGroupHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	org, orgerr := orgloader.Get(vars["org"])
	if orgerr != nil {
		jsonErrorReport(w, r, orgerr.Error(), orgerr.Status())
		return
	}

	opUser, oerr := reqctx.CtxReqUser(r.Context())
	if oerr != nil {
		jsonErrorReport(w, r, oerr.Error(), oerr.Status())
		return
	}
}
*/
