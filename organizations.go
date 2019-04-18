/*
 * Copyright (c) 2013-2019, Jeremy Bingham (<jbingham@gmail.com>)
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
	"fmt"
	"github.com/ctdk/goiardi/actor"
	"github.com/ctdk/goiardi/association"
	"github.com/ctdk/goiardi/client"
	"github.com/ctdk/goiardi/container"
	"github.com/ctdk/goiardi/environment"
	"github.com/ctdk/goiardi/group"
	"github.com/ctdk/goiardi/organization"
	"github.com/ctdk/goiardi/orgloader"
	"github.com/ctdk/goiardi/user"
	"github.com/ctdk/goiardi/util"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"regexp"
)

// might also be best split up
func orgToolHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)
	pathArray := splitPath(r.URL.Path)
	orgName := vars["org"]

	// Otherwise, it's org work.
	var orgResponse interface{}

	op := pathArray[2]
	org, err := orgloader.Get(orgName)
	if err != nil {
		log.Printf("Huh? err is %v", err)
		jsonErrorReport(w, r, err.Error(), err.Status())
		return
	}
	opUser, oerr := actor.GetReqUser(org, r.Header.Get("X-OPS-USERID"))
	_ = opUser
	if oerr != nil {
		jsonErrorReport(w, r, oerr.Error(), oerr.Status())
		return
	}

	if f, ferr := org.PermCheck.RootCheckPerm(opUser, "update"); ferr != nil {
		jsonErrorReport(w, r, ferr.Error(), ferr.Status())
		return
	} else if !f {
		jsonErrorReport(w, r, "You do not have permission to do that", http.StatusForbidden)
		return
	}

	switch op {
	case "_validator_key":
		if r.Method == "POST" {
			valname := util.JoinStr(org.Name, "-validator")
			val, err := client.Get(org, valname)
			if err != nil {
				jsonErrorReport(w, r, err.Error(), err.Status())
				return
			}
			pem, perr := val.GenerateKeys()
			if perr != nil {
				jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
				return
			}
			oR := make(map[string]interface{})
			oR["private_key"] = pem
			orgResponse = oR
		} else {
			jsonErrorReport(w, r, "Unrecognized method", http.StatusMethodNotAllowed)
			return
		}
	case "association_requests":
		if len(pathArray) == 4 {
			if r.Method != "DELETE" {
				jsonErrorReport(w, r, "unrecognized method", http.StatusMethodNotAllowed)
				return
			}
			id := vars["id"]
			re := regexp.MustCompile(util.JoinStr("(.+)-", orgName))
			userChk := re.FindStringSubmatch(id)
			if userChk == nil {
				util.JSONErrorReport(w, r, util.JoinStr("Invalid ID ", id, ". Must be of the form username-", orgName), http.StatusNotFound)
				return
			}
			log.Printf("Deleting assoc req %s", id)
			// Looks like this is supposed to be a delete.
			ar, err := association.GetReq(id)
			if err != nil {
				jsonErrorNonArrayReport(w, r, err.Error(), err.Status())
				return
			}
			err = ar.Delete()
			if err != nil {
				jsonErrorReport(w, r, err.Error(), err.Status())
				return
			}
			log.Printf("deleted id %s", id)
			ur, _ := association.GetAllUsersAssociationReqs(org)
			log.Printf("assoc reqs after are:")
			for _, v := range ur {
				log.Printf("%s", v.Key())
			}

			oR := make(map[string]interface{})
			oR["id"] = id
			oR["username"] = userChk[1]
			orgResponse = oR
		} else {
			switch r.Method {
			case "GET":
				// returns a list of associations with
				// this org.
				userReqs, err := association.GetAllUsersAssociationReqs(org)
				log.Printf("user association request length: %d", len(userReqs))
				log.Printf("User association requests returned: %+v", userReqs)
				log.Println("the association reqs")
				for _, v := range userReqs {
					log.Printf("%+v", v)
					log.Printf("%s", v.Key())
				}
				if err != nil {
					jsonErrorReport(w, r, err.Error(), err.Status())
					return
				}
				oR := make([]map[string]interface{}, len(userReqs))
				for i, ua := range userReqs {
					m := make(map[string]interface{})
					m["id"] = ua.Key()
					m["username"] = ua.User.Name
					oR[i] = m
				}
				orgResponse = oR

			case "POST":
				// creates the association.
				arData, jerr := parseObjJSON(r.Body)
				if jerr != nil {
					jsonErrorReport(w, r, jerr.Error(), http.StatusBadRequest)
					return
				}
				userName, ok := arData["user"].(string)
				if !ok {
					jsonErrorReport(w, r, "user name missing or invalid", http.StatusBadRequest)
					return
				}
				user, err := user.Get(userName)
				if err != nil {
					jsonErrorReport(w, r, err.Error(), err.Status())
					return
				}
				assoc, err := association.SetReq(user, org, opUser)
				if err != nil {
					jsonErrorNonArrayReport(w, r, err.Error(), err.Status())
					return
				}
				w.WriteHeader(http.StatusCreated)
				oR := make(map[string]interface{})
				oR["uri"] = util.CustomURL(util.JoinStr(r.URL.Path, "/", assoc.Key()))
				orgResponse = oR
			default:
				jsonErrorReport(w, r, "Unrecognized method", http.StatusMethodNotAllowed)
				return
			}
		}
	default:
		jsonErrorReport(w, r, "Unknown organization endpoint, rather unlikely to reach", http.StatusBadRequest)
		return
	}
	enc := json.NewEncoder(w)
	if err := enc.Encode(&orgResponse); err != nil {
		jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
	}
}

func orgMainHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)
	orgName := vars["org"]
	org, err := orgloader.Get(orgName)
	var orgResponse map[string]interface{}

	if err != nil {
		jsonErrorReport(w, r, err.Error(), err.Status())
		return
	}
	opUser, oerr := actor.GetReqUser(org, r.Header.Get("X-OPS-USERID"))
	if oerr != nil {
		jsonErrorReport(w, r, oerr.Error(), oerr.Status())
		return
	}
	if f, ferr := org.PermCheck.RootCheckPerm(opUser, "read"); ferr != nil {
		jsonErrorReport(w, r, ferr.Error(), ferr.Status())
		return
	} else if !f {
		jsonErrorReport(w, r, "You do not have permission to do that", http.StatusForbidden)
		return
	}

	switch r.Method {
	case "GET", "DELETE":
		orgResponse = org.ToJSON()
		if r.Method == "DELETE" {
			if f, ferr := org.PermCheck.RootCheckPerm(opUser, "delete"); ferr != nil {
				jsonErrorReport(w, r, ferr.Error(), ferr.Status())
				return
			} else if !f {
				jsonErrorReport(w, r, "You do not have permission to do that", http.StatusForbidden)
				return
			}
			err := org.Delete()
			if err != nil {
				jsonErrorReport(w, r, err.Error(), err.Status())
				return
			}
		}
	case "PUT":
		if f, ferr := org.PermCheck.RootCheckPerm(opUser, "update"); ferr != nil {
			jsonErrorReport(w, r, ferr.Error(), ferr.Status())
			return
		} else if !f {
			jsonErrorReport(w, r, "You do not have permission to do that", http.StatusForbidden)
			return
		}
		orgData, jerr := parseObjJSON(r.Body)
		if jerr != nil {
			jsonErrorReport(w, r, jerr.Error(), http.StatusBadRequest)
			return
		}
		name, verr := util.ValidateAsString(orgData["name"])
		if verr != nil {
			jsonErrorReport(w, r, "field name missing or invalid", http.StatusBadRequest)
			return
		}
		fullName, verr := util.ValidateAsString(orgData["full_name"])
		if verr != nil {
			jsonErrorReport(w, r, "field full name missing or invalid", http.StatusBadRequest)
			return
		}
		if name != org.Name {
			jsonErrorReport(w, r, "Field 'name' invalid", http.StatusBadRequest)
			return
		}
		org.FullName = fullName
		err := org.Save()
		if err != nil {
			jsonErrorReport(w, r, err.Error(), err.Status())
			return
		}
		orgResponse = org.ToJSON()
		// Crazy update-only thing - if it sends an 'org_type' field,
		// even though erchef (and by extension goiardi) don't use it,
		// the tooling seems to expect it to come back. Uh, ok.
		if _, ok := orgData["org_type"]; ok {
			orgResponse["org_type"] = orgData["org_type"]
		}
	default:
		jsonErrorReport(w, r, "Unrecognized method", http.StatusMethodNotAllowed)
		return
	}
	enc := json.NewEncoder(w)
	if err := enc.Encode(&orgResponse); err != nil {
		jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
	}
}

func orgHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var orgResponse map[string]interface{}

	opUser, oerr := actor.GetReqUser(nil, r.Header.Get("X-OPS-USERID"))
	if oerr != nil {
		jsonErrorReport(w, r, oerr.Error(), oerr.Status())
		return
	}
	// try the default org for this
	orgDef, err := orgloader.Get("default")
	if err != nil {
		jsonErrorReport(w, r, err.Error(), err.Status())
		return
	}
	if f, ferr := orgDef.PermCheck.RootCheckPerm(opUser, "read"); ferr != nil {
		jsonErrorReport(w, r, ferr.Error(), ferr.Status())
		return
	} else if !f {
		jsonErrorReport(w, r, "You do not have permission to do that", http.StatusForbidden)
		return
	}
	switch r.Method {
	case "GET":
		orgList := organization.GetList()
		orgResponse = make(map[string]interface{})
		for _, o := range orgList {
			itemURL := fmt.Sprintf("/organizations/%s", o)
			orgResponse[o] = util.CustomURL(itemURL)
		}
	case "POST":
		if f, ferr := orgDef.PermCheck.RootCheckPerm(opUser, "create"); ferr != nil {
			jsonErrorReport(w, r, ferr.Error(), ferr.Status())
			return
		} else if !f {
			jsonErrorReport(w, r, "You do not have permission to do that", http.StatusForbidden)
			return
		}
		orgData, jerr := parseObjJSON(r.Body)
		if jerr != nil {
			jsonErrorReport(w, r, jerr.Error(), http.StatusBadRequest)
			return
		}
		orgName, verr := util.ValidateAsString(orgData["name"])
		if verr != nil {
			jsonErrorReport(w, r, "field name missing or invalid", http.StatusBadRequest)
			return
		}
		orgFullName, verr := util.ValidateAsString(orgData["full_name"])
		if verr != nil {
			jsonErrorReport(w, r, "field full name missing or invalid", http.StatusBadRequest)
			return
		}
		org, err := orgloader.New(orgName, orgFullName)
		if err != nil {
			jsonErrorReport(w, r, err.Error(), err.Status())
			return
		}
		validator, pem, err := makeValidator(org, opUser)
		if err != nil {
			jsonErrorReport(w, r, err.Error(), err.Status())
			return
		}
		cerr := container.MakeDefaultContainers(org)
		if cerr != nil {
			jsonErrorReport(w, r, cerr.Error(), cerr.Status())
			return
		}
		environment.MakeDefaultEnvironment(org)
		gerr := group.MakeDefaultGroups(org)
		if gerr != nil {
			jsonErrorReport(w, r, gerr.Error(), gerr.Status())
			return
		}
		clientGroup, err := group.Get(org, "clients")
		if err != nil {
			jsonErrorReport(w, r, err.Error(), err.Status())
			return
		}
		err = clientGroup.AddActor(validator)
		if err != nil {
			jsonErrorReport(w, r, err.Error(), err.Status())
			return
		}
		err = clientGroup.Save()
		if err != nil {
			jsonErrorReport(w, r, err.Error(), err.Status())
			return
		}
		orgResponse = org.ToJSON()
		orgResponse["private_key"] = pem
		orgResponse["clientname"] = validator.Name
		orgResponse["uri"] = util.CustomURL(util.JoinStr("/organizations/", org.Name))
		w.WriteHeader(http.StatusCreated)
	default:
		jsonErrorReport(w, r, "Unrecognized method", http.StatusMethodNotAllowed)
		return
	}
	enc := json.NewEncoder(w)
	if err := enc.Encode(&orgResponse); err != nil {
		jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
	}
}

func makeValidator(org *organization.Organization, opUser actor.Actor) (*client.Client, string, util.Gerror) {
	valname := util.JoinStr(org.Name, "-validator")
	val, err := client.New(org, valname)
	if err != nil {
		return nil, "", err
	}
	val.Validator = true
	pem, perr := val.GenerateKeys()
	if perr != nil {
		return nil, "", util.CastErr(perr)
	}
	perr = val.Save()
	if perr != nil {
		return nil, "", util.CastErr(perr)
	}

	// Unusually, set creator perms for this particular validator to
	// "pivotal". It's special because it's not created the usual way, but
	// as part of creating an organization. If we don't do this it'll go
	// and return the wrong actors in the ACL. (Arrrrgh.)
	if aerr := org.PermCheck.CreatorOnly(val, opUser); aerr != nil {
		return nil, "", aerr
	}

	return val, pem, nil
}
