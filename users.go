/* User handler functions */

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

package main

import (
	"encoding/json"
	"fmt"
	"github.com/ctdk/goiardi/actor"
	"github.com/ctdk/goiardi/association"
	"github.com/ctdk/goiardi/group"
	"github.com/ctdk/goiardi/loginfo"
	"github.com/ctdk/goiardi/masteracl"
	"github.com/ctdk/goiardi/organization"
	"github.com/ctdk/goiardi/orgloader"
	"github.com/ctdk/goiardi/reqctx"
	"github.com/ctdk/goiardi/user"
	"github.com/ctdk/goiardi/util"
	"github.com/gorilla/mux"
	"github.com/tideland/golib/logger"
	"net/http"
	"strconv"
	"strings"
)

func userHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)
	userName := vars["name"]

	var org *organization.Organization
	if _, ok := vars["org"]; ok {
		var orgerr util.Gerror
		org, orgerr = reqctx.CtxOrg(r.Context())
		if orgerr != nil {
			jsonErrorReport(w, r, orgerr.Error(), orgerr.Status())
			return
		}
	}

	opUser, oerr := reqctx.CtxReqUser(r.Context())

	if oerr != nil {
		jsonErrorReport(w, r, oerr.Error(), oerr.Status())
		return
	}

	switch r.Method {
	case http.MethodHead:
		permCheck := func(r *http.Request, userName string, opUser actor.Actor) util.Gerror {
			if f, ferr := masteracl.MasterCheckPerm(opUser, masteracl.Users, "read"); ferr != nil {
				return ferr
			} else if !f {
				chefUser, err := user.Get(userName)
				if err != nil {
					return err
				}
				if !opUser.IsSelf(chefUser) {
					err = util.Errorf("not same")
					err.SetStatus(http.StatusForbidden)
					return err
				}
			}

			return nil
		}
		headChecking(w, r, opUser, org, userName, user.DoesExist, permCheck)
		return
	case http.MethodDelete:
		chefUser, err := user.Get(userName)
		if err != nil {
			jsonErrorReport(w, r, err.Error(), http.StatusNotFound)
			return
		}
		if !opUser.IsSelf(chefUser) {
			if f, ferr := masteracl.MasterCheckPerm(opUser, masteracl.Users, "delete"); ferr != nil {
				jsonErrorReport(w, r, ferr.Error(), ferr.Status())
			} else if !f {
				jsonErrorReport(w, r, "Deleting that user is forbidden", http.StatusForbidden)
			}

			return
		}
		/* Docs were incorrect. It does want the body of the
		 * deleted object. */
		jsonUser := chefUser.ToJSON()

		// Clear this user USAGs, groups and org associations if any
		// remain.
		orgs, err := association.OrgAssociations(chefUser)
		if err != nil {
			jsonErrorReport(w, r, err.Error(), err.Status())
			return
		}
		err = association.DelAllUserAssocReqs(chefUser)
		if err != nil {
			jsonErrorReport(w, r, err.Error(), err.Status())
			return
		}
		err = association.DelAllUserAssociations(chefUser)
		if err != nil {
			jsonErrorReport(w, r, err.Error(), err.Status())
			return
		}
		go func(orgs []*organization.Organization, chefUser *user.User) {
			for _, o := range orgs {
				group.ClearActor(o, chefUser)
				o.PermCheck.RemoveUser(chefUser)
				usagName := fmt.Sprintf("%x", []byte(chefUser.Username))
				if usag, _ := group.Get(o, usagName); usag != nil {
					usag.Delete() // fire and forget
				}
			}
		}(orgs, chefUser)

		/* Log the delete event *before* deleting the user, in
		 * case the user is deleting itself. */
		if lerr := loginfo.LogEvent(org, opUser, chefUser, "delete"); lerr != nil {
			jsonErrorReport(w, r, lerr.Error(), http.StatusInternalServerError)
			return
		}
		err = chefUser.Delete()
		if err != nil {
			jsonErrorReport(w, r, err.Error(), http.StatusForbidden)
			return
		}
		enc := json.NewEncoder(w)
		if encerr := enc.Encode(&jsonUser); encerr != nil {
			jsonErrorReport(w, r, encerr.Error(), http.StatusInternalServerError)
			return
		}
	case http.MethodGet:
		chefUser, err := user.Get(userName)

		if err != nil {
			jsonErrorReport(w, r, err.Error(), http.StatusNotFound)
			return
		}
		if !opUser.IsSelf(chefUser) {
			if f, ferr := masteracl.MasterCheckPerm(opUser, masteracl.Users, "read"); ferr != nil {
				jsonErrorReport(w, r, ferr.Error(), ferr.Status())
			} else if !f {
				orgAdmin, oerr := isOrgAdminForUser(chefUser, opUser)
				if oerr != nil {
					jsonErrorReport(w, r, oerr.Error(), oerr.Status())
					return
				}
				if !orgAdmin {
					jsonErrorReport(w, r, "You are not allowed to perform that action.", http.StatusForbidden)
					return
				}
			}
		}

		/* API docs are wrong here re: public_key vs.
		 * certificate. Also orgname (at least w/ open source)
		 * and clientname, and it wants chef_type and
		 * json_class
		 */
		jsonUser := chefUser.ToJSON()
		enc := json.NewEncoder(w)
		if encerr := enc.Encode(&jsonUser); encerr != nil {
			jsonErrorReport(w, r, encerr.Error(), http.StatusInternalServerError)
			return
		}
	case http.MethodPut:
		userData, jerr := parseObjJSON(r.Body)
		if jerr != nil {
			jsonErrorReport(w, r, jerr.Error(), http.StatusBadRequest)
			return
		}
		chefUser, err := user.Get(userName)
		if err != nil {
			jsonErrorReport(w, r, err.Error(), http.StatusNotFound)
			return
		}

		/* Makes chef-pedant happy. I suppose it is, after all,
		 * pedantic. */
		if averr := util.CheckAdminPlusValidator(userData); averr != nil {
			jsonErrorReport(w, r, averr.Error(), averr.Status())
			return
		}

		f, ferr := masteracl.MasterCheckPerm(opUser, masteracl.Users, "update")
		if ferr != nil {
			jsonErrorReport(w, r, ferr.Error(), ferr.Status())
		}
		if !f && !opUser.IsSelf(chefUser) {
			jsonErrorReport(w, r, "You are not allowed to perform that action.", http.StatusForbidden)
			return
		}
		if !f {
			aerr := opUser.CheckPermEdit(userData, "admin")
			if aerr != nil {
				jsonErrorReport(w, r, aerr.Error(), aerr.Status())
				return
			}
		}

		var nameFromJSON interface{}
		var ok bool
		if nameFromJSON, ok = userData["username"]; !ok {
			nameFromJSON, _ = userData["name"]
		}

		jsonName, sterr := util.ValidateAsString(nameFromJSON)
		if sterr != nil {
			jsonErrorReport(w, r, sterr.Error(), http.StatusBadRequest)
			return
		}

		/* If userName and userData["username"] aren't the
		 * same, we're renaming. Check the new name doesn't
		 * already exist. */
		jsonUser := chefUser.ToJSON()
		delete(jsonUser, "public_key")
		if userName != jsonName {
			oldACLName := chefUser.ACLName()
			err := chefUser.Rename(jsonName)
			if err != nil {
				jsonErrorReport(w, r, err.Error(), err.Status())
				return
			}
			err = chefUser.Save()
			if err != nil {
				jsonErrorReport(w, r, err.Error(), err.Status())
				return
			}
			orgs, err := association.OrgAssociations(chefUser)
			if err != nil {
				jsonErrorReport(w, r, err.Error(), err.Status())
				return
			}
			go func(orgs []*organization.Organization, chefUser *user.User) {
				for _, o := range orgs {
					o.PermCheck.RenameMember(chefUser, oldACLName)
				}
			}(orgs, chefUser)
			w.WriteHeader(http.StatusCreated)
			// looks like we need to send back a non-standard
			// response for renaming :-/
			renameResp := make(map[string]string)
			renameResp["uri"] = util.CustomURL(util.JoinStr("/users/", chefUser.Username))
			enc := json.NewEncoder(w)
			if encerr := enc.Encode(&renameResp); encerr != nil {
				jsonErrorReport(w, r, encerr.Error(), http.StatusInternalServerError)
				return
			}
			return
		}
		if uerr := chefUser.UpdateFromJSON(userData); uerr != nil {
			jsonErrorReport(w, r, uerr.Error(), uerr.Status())
			return
		}

		var pkChange bool
		if pk, pkfound := userData["public_key"]; pkfound {
			switch pk := pk.(type) {
			case string:
				if strings.Contains(pk, "CERTIFICATE") {
					logger.Infof("Tried to set the public key for user %s to be a certificate", chefUser.Username)
					p, _ := userData["private_key"].(bool)
					if !p {
						jsonErrorReport(w, r, "invalid public key (is a certificate)", http.StatusBadRequest)
						return
					}
					logger.Infof("going to recreate private key")
				} else {
					if pkok, pkerr := user.ValidatePublicKey(pk); !pkok {
						jsonErrorReport(w, r, pkerr.Error(), http.StatusBadRequest)
						return
					}
					chefUser.SetPublicKey(pk)
					jsonUser["public_key"] = pk
					pkChange = true
				}
			case nil:
				//show_public_key = false

			default:
				jsonErrorReport(w, r, "Bad request", http.StatusBadRequest)
				return
			}
		}

		var privChange bool
		if p, pfound := userData["private_key"]; pfound {
			switch p := p.(type) {
			case bool:
				if p {
					var perr error
					if jsonUser["private_key"], perr = chefUser.GenerateKeys(); perr != nil {
						jsonErrorReport(w, r, perr.Error(), http.StatusInternalServerError)
						return
					}
					// make sure the json
					// client gets the new
					// public key
					jsonUser["public_key"] = chefUser.PublicKey()
					privChange = true
					pkChange = true
				}
			default:
				jsonErrorReport(w, r, "Bad request", http.StatusBadRequest)
				return
			}
		}

		serr := chefUser.Save()
		if serr != nil {
			jsonErrorReport(w, r, serr.Error(), serr.Status())
			return
		}
		if lerr := loginfo.LogEvent(org, opUser, chefUser, "modify"); lerr != nil {
			jsonErrorReport(w, r, lerr.Error(), http.StatusInternalServerError)
			return
		}

		if pkChange {
			renameResp := make(map[string]string)
			renameResp["uri"] = util.CustomURL(util.JoinStr("/users/", chefUser.Username))
			if privChange {
				renameResp["private_key"] = jsonUser["private_key"].(string)
			}
			enc := json.NewEncoder(w)
			if encerr := enc.Encode(&renameResp); encerr != nil {
				jsonErrorReport(w, r, encerr.Error(), http.StatusInternalServerError)
				return
			}
			return
		}
		enc := json.NewEncoder(w)
		if encerr := enc.Encode(&jsonUser); encerr != nil {
			jsonErrorReport(w, r, encerr.Error(), http.StatusInternalServerError)
			return
		}
	default:
		jsonErrorReport(w, r, "Unrecognized method for user!", http.StatusMethodNotAllowed)
	}
}

func userListHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userResponse := make(map[string]interface{})
	opUser, oerr := actor.GetReqUser(nil, r.Header.Get("X-OPS-USERID"))
	if oerr != nil {
		jsonErrorReport(w, r, oerr.Error(), oerr.Status())
		return
	}
	vars := mux.Vars(r)
	var org *organization.Organization
	if orgName, ok := vars["org"]; ok {
		var orgerr util.Gerror
		org, orgerr = orgloader.Get(orgName)
		if orgerr != nil {
			jsonErrorReport(w, r, orgerr.Error(), orgerr.Status())
			return
		}
	}
	r.ParseForm()
	var email string
	if e, found := r.Form["email"]; found {
		if len(e) < 0 {
			jsonErrorReport(w, r, "invalid email param for search", http.StatusBadRequest)
			return
		}
		email = e[0]
	}
	var verbose bool
	if v, found := r.Form["verbose"]; found {
		if len(v) < 0 {
			jsonErrorReport(w, r, "invalid verbosity", http.StatusBadRequest)
			return
		}
		vb, err := strconv.ParseBool(v[0])
		if err != nil {
			jsonErrorReport(w, r, err.Error(), http.StatusBadRequest)
			return
		}
		verbose = vb
	}

	switch r.Method {
	case http.MethodGet:
		// Seems the user has to be a superuser for this functionality
		// now.
		if f, ferr := masteracl.MasterCheckPerm(opUser, masteracl.Users, "read"); ferr != nil {
			jsonErrorReport(w, r, ferr.Error(), ferr.Status())
		} else if !f {
			jsonErrorReport(w, r, "You are not allowed to take this action.", http.StatusForbidden)
			return
		}

		if email == "" {
			if verbose {
				users := user.AllUsers()
				for _, u := range users {
					userResponse[u.Username] = u.ToJSON()
				}
			} else {
				userList := user.GetList()
				for _, k := range userList {
					itemURL := util.JoinStr("/users/", k)
					userResponse[k] = util.CustomURL(itemURL)
				}
			}
		} else {
			u, _ := user.GetByEmail(email)
			if u != nil {
				if verbose {
					userResponse[u.Username] = u.ToJSON()
				} else {
					itemURL := util.JoinStr("/users/", u.Username)
					userResponse[u.Username] = util.CustomURL(itemURL)
				}
			}
		}
	case http.MethodPost:
		userData, jerr := parseObjJSON(r.Body)
		if jerr != nil {
			jsonErrorReport(w, r, jerr.Error(), http.StatusBadRequest)
			return
		}
		if averr := util.CheckAdminPlusValidator(userData); averr != nil {
			jsonErrorReport(w, r, averr.Error(), averr.Status())
			return
		}
		if f, ferr := masteracl.MasterCheckPerm(opUser, masteracl.Users, "update"); ferr != nil {
			jsonErrorReport(w, r, ferr.Error(), ferr.Status())
		} else if !f {
			if opUser.IsValidator() {
				if aerr := opUser.CheckPermEdit(userData, "admin"); aerr != nil {
					jsonErrorReport(w, r, aerr.Error(), aerr.Status())
					return
				}
				if verr := opUser.CheckPermEdit(userData, "validator"); verr != nil {
					jsonErrorReport(w, r, verr.Error(), verr.Status())
					return
				}

			} else {
				jsonErrorReport(w, r, "You are not allowed to take this action.", http.StatusForbidden)
				return
			}
		}

		var nameFromJSON interface{}
		var ok bool
		if nameFromJSON, ok = userData["username"]; !ok {
			nameFromJSON, _ = userData["name"]
		}
		userName, sterr := util.ValidateAsString(nameFromJSON)
		if sterr != nil || userName == "" {
			err := fmt.Errorf("Field 'name' missing")
			jsonErrorReport(w, r, err.Error(), http.StatusBadRequest)
			return
		}

		chefUser, err := user.NewFromJSON(userData)
		if err != nil {
			jsonErrorReport(w, r, err.Error(), err.Status())
			return
		}

		if publicKey, pkok := userData["public_key"]; !pkok {
			var perr error
			if userResponse["private_key"], perr = chefUser.GenerateKeys(); perr != nil {
				jsonErrorReport(w, r, perr.Error(), http.StatusInternalServerError)
				return
			}
		} else {
			switch publicKey := publicKey.(type) {
			case string:
				if pkok, pkerr := user.ValidatePublicKey(publicKey); !pkok {
					jsonErrorReport(w, r, pkerr.Error(), pkerr.Status())
					return
				}
				chefUser.SetPublicKey(publicKey)
			case nil:

				var perr error
				if userResponse["private_key"], perr = chefUser.GenerateKeys(); perr != nil {
					jsonErrorReport(w, r, perr.Error(), http.StatusInternalServerError)
					return
				}
			default:
				jsonErrorReport(w, r, "Bad public key", http.StatusBadRequest)
				return
			}
		}
		/* If we make it here, we want the public key in the
		 * response. I think. Maybe not anymore, though. */
		//userResponse["public_key"] = chefUser.PublicKey()

		chefUser.Save()
		if lerr := loginfo.LogEvent(org, opUser, chefUser, "create"); lerr != nil {
			jsonErrorReport(w, r, lerr.Error(), http.StatusInternalServerError)
			return
		}
		userResponse["uri"] = util.CustomURL(util.JoinStr("/users/", chefUser.Username))
		w.WriteHeader(http.StatusCreated)
	default:
		jsonErrorReport(w, r, "Method not allowed for clients or users", http.StatusMethodNotAllowed)
		return
	}
	enc := json.NewEncoder(w)
	if err := enc.Encode(&userResponse); err != nil {
		jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
	}
}

func userListOrgHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)
	userName := vars["name"]

	if r.Method != http.MethodGet {
		jsonErrorReport(w, r, "unrecognized method", http.StatusMethodNotAllowed)
		return
	}

	opUser, oerr := actor.GetReqUser(nil, r.Header.Get("X-OPS-USERID"))
	if oerr != nil {
		jsonErrorReport(w, r, oerr.Error(), oerr.Status())
		return
	}

	chefUser, err := user.Get(userName)

	if !opUser.IsSelf(chefUser) {
		if f, ferr := masteracl.MasterCheckPerm(opUser, masteracl.Users, "read"); ferr != nil {
			jsonErrorReport(w, r, ferr.Error(), ferr.Status())
		} else if !f {
			ook, err := isOrgAdminForUser(chefUser, opUser)
			if err != nil {
				jsonErrorReport(w, r, err.Error(), err.Status())
				return
			}
			if !ook {
				jsonErrorReport(w, r, "you are not allowed to perform that action", http.StatusForbidden)
				return
			}
		}
	}

	if err != nil {
		jsonErrorReport(w, r, err.Error(), err.Status())
		return
	}
	if err != nil {
		jsonErrorReport(w, r, err.Error(), err.Status())
		return
	}

	orgs, err := association.OrgAssociations(chefUser)

	response := make([]map[string]interface{}, len(orgs))
	for i, o := range orgs {
		or := map[string]interface{}{"organization": o.ToJSON()}
		response[i] = or
	}

	enc := json.NewEncoder(w)
	if err := enc.Encode(&response); err != nil {
		jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
	}
}

func isOrgAdminForUser(chkUser *user.User, opUser actor.Actor) (bool, util.Gerror) {
	// Another operation that may well be significantly easier when it's
	// DB-ified.
	orgs, err := association.OrgAssociations(chkUser)
	if err != nil {
		return false, err
	}
	logger.Debugf("number of orgs for %s: %d", chkUser.Username, len(orgs))
	for _, org := range orgs {
		logger.Debugf("in org %s", org.Name)
		admin, err := group.Get(org, "admins")
		// unlikely
		if err != nil {
			return false, err
		}
		logger.Debugf("still here. Number of actors? %d", len(admin.Actors))
		logger.Debugf("wtf do we think the group is: %+v", admin)
		for _, aa := range admin.Actors {
			logger.Debugf("%s", aa.GetName())
		}
		if admin.SeekActor(opUser) {
			logger.Debugf("user %s is an admin in %s", opUser.GetName(), org.Name)
			return true, nil
		}
	}
	return false, nil
}
