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
	"github.com/ctdk/goiardi/organization"
	"github.com/ctdk/goiardi/orgloader"
	"github.com/ctdk/goiardi/policy"
	"github.com/ctdk/goiardi/reqctx"
	"github.com/ctdk/goiardi/util"
	"github.com/gorilla/mux"
	"net/http"
)

const (
	groupListLen = 1
	groupDetailLen = 2
	groupPolicyDetailLen = 4
)

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

	pathArray := splitPath(r.URL.Path)[2:]
	pgName := vars["name"]
	policyName := vars["policy"]

	// cause I would rather switch.......
	var err util.Gerror

	switch len(pathArray) {
	case groupListLen:
		err = policyGroupList(w, r, org, opUser)
	case groupDetailLen:
		err = policyGroupDetail(w, r, org, opUser, pgName)
	case groupPolicyDetailLen:
		err = policyGroupPolicyDetail(w, r, org, opUser, pgName, policyName)
	default:
		jsonErrorReport(w, r, "Not sure how we got here, but this URL has an improper number of elements.", http.StatusBadRequest)
		return
	}
	if err != nil {
		jsonErrorReport(w, r, err.Error(), err.Status())
	}
	return
}

func policyGroupList(w http.ResponseWriter, r *http.Request, org *organization.Organization, opUser actor.Actor) util.Gerror {
	if f, ferr := org.PermCheck.CheckContainerPerm(opUser, "policies", "read"); ferr != nil {
		return ferr
	} else if !f {
		err := util.Errorf("You do not have permission to do that")
		err.SetStatus(http.StatusForbidden)
		return err
	}

	// HEAD check
	if r.Method == http.MethodHead {
		headResponse(w, r, http.StatusOK)
		return nil
	}

	if r.Method != http.MethodGet {
		err := util.Errorf("Method not allowed")
		err.SetStatus(http.StatusMethodNotAllowed)
		return err
	}

	groupList := make(map[string]map[string]interface{})
	allPg, err := policy.GetAllPolicyGroups(org)
	if err != nil {
		return err
	}

	for _, pg := range allPg {
		pgr := make(map[string]interface{})
		pgr["uri"] = pg.URI()
		pgr["policies"] = pg.GetPolicyMap()
		groupList[pg.Name] = pgr
	}

	enc := json.NewEncoder(w)
	if encErr := enc.Encode(&groupList); encErr != nil {
		cErr := util.CastErr(encErr)
		cErr.SetStatus(http.StatusInternalServerError)
		return cErr
	}

	return nil
}

func policyGroupDetail(w http.ResponseWriter, r *http.Request, org *organization.Organization, opUser actor.Actor, pgName string) util.Gerror {
	// try reducing the number of perm checks by not checking for delete
	// separate from read. Instead, set the relevant permission depending
	// on the method.
	//
	// If it doesn't work, of course, just set this back and add the
	// separate delete perm check back in below.
	var perm string
	if r.Method == http.MethodDelete {
		perm = "delete"
	} else {
		perm = "read"
	}

	if f, ferr := org.PermCheck.CheckContainerPerm(opUser, "policies", perm); ferr != nil {
		return ferr
	} else if !f {
		err := util.Errorf("You do not have permission to do that")
		err.SetStatus(http.StatusForbidden)
		return err
	}

	switch r.Method {
	case http.MethodHead:
		permCheck := func(r *http.Request, policyName string, opUser actor.Actor) util.Gerror {
			if f, ferr := org.PermCheck.CheckContainerPerm(opUser, "policies", "read"); ferr != nil {
				return ferr
			} else if !f {
				return headForbidden()
			}
			return nil
		}

		headChecking(w, r, opUser, org, pgName, policy.DoesPolicyGroupExist, permCheck)
		return nil
	case http.MethodGet, http.MethodDelete:
		pg, err := policy.GetPolicyGroup(org, pgName)
		if err != nil {
			return err
		}
		
		pgr := make(map[string]interface{})
		pgr["uri"] = pg.URI()
		pgr["policies"] = pg.GetPolicyMap()
		if r.Method == http.MethodDelete {
			if err = pg.Delete(); err != nil {
				return err
			}
		}
	default:
		err := util.Errorf("Method not allowed")
		err.SetStatus(http.StatusMethodNotAllowed)
		return err
	}

	enc := json.NewEncoder(w)
	if encErr := enc.Encode(&pgr); encErr != nil {
		cErr := util.CastErr(encErr)
		cErr.SetStatus(http.StatusInternalServerError)
		return cErr
	}

	return nil
}

func policyGroupPolicyDetail(w http.ResponseWriter, r *http.Request, org *organization.Organization, opUser actor.Actor, pgName string, policyName string) util.Gerror {
	if f, ferr := org.PermCheck.CheckContainerPerm(opUser, "policies", "read"); ferr != nil {
		return ferr
	} else if !f {
		err := util.Errorf("You do not have permission to do that")
		err.SetStatus(http.StatusForbidden)
		return err
	}

	return nil
}
