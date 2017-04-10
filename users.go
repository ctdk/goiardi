/* User handler functions */

/*
 * Copyright (c) 2013-2016, Jeremy Bingham (<jeremy@goiardi.gl>)
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
	"github.com/ctdk/goiardi/loginfo"
	"github.com/ctdk/goiardi/user"
	"github.com/ctdk/goiardi/util"
	"net/http"
)

func userHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	path := splitPath(r.URL.Path)
	userName := path[1]
	opUser, oerr := actor.GetReqUser(r.Header.Get("X-OPS-USERID"))
	if oerr != nil {
		jsonErrorReport(w, r, oerr.Error(), oerr.Status())
		return
	}

	switch r.Method {
	case http.MethodHead:
		permCheck := func(r *http.Request, userName string, opUser actor.Actor) util.Gerror {
			if !opUser.IsAdmin() {
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
		headChecking(w, r, opUser, userName, user.DoesExist, permCheck)
		return
	case http.MethodDelete:
		chefUser, err := user.Get(userName)
		if err != nil {
			jsonErrorReport(w, r, err.Error(), err.Status())
			return
		}
		if !opUser.IsAdmin() && !opUser.IsSelf(chefUser) {
			jsonErrorReport(w, r, "Deleting that user is forbidden", http.StatusForbidden)
			return
		}
		/* Docs were incorrect. It does want the body of the
		 * deleted object. */
		jsonUser := chefUser.ToJSON()

		/* Log the delete event *before* deleting the user, in
		 * case the user is deleting itself. */
		if lerr := loginfo.LogEvent(opUser, chefUser, "delete"); lerr != nil {
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
			jsonErrorReport(w, r, err.Error(), err.Status())
			return
		}
		if !opUser.IsAdmin() && !opUser.IsSelf(chefUser) {
			jsonErrorReport(w, r, "You are not allowed to perform that action.", http.StatusForbidden)
			return
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

		if !opUser.IsAdmin() && !opUser.IsSelf(chefUser) {
			jsonErrorReport(w, r, "You are not allowed to perform that action.", http.StatusForbidden)
			return
		}
		if !opUser.IsAdmin() {
			aerr := opUser.CheckPermEdit(userData, "admin")
			if aerr != nil {
				jsonErrorReport(w, r, aerr.Error(), aerr.Status())
				return
			}
		}

		jsonName, sterr := util.ValidateAsString(userData["name"])
		if sterr != nil {
			jsonErrorReport(w, r, sterr.Error(), http.StatusBadRequest)
			return
		}

		/* If userName and userData["name"] aren't the
		 * same, we're renaming. Check the new name doesn't
		 * already exist. */
		jsonUser := chefUser.ToJSON()
		delete(jsonUser, "public_key")
		if userName != jsonName {
			err := chefUser.Rename(jsonName)
			if err != nil {
				jsonErrorReport(w, r, err.Error(), err.Status())
				return
			}
			w.WriteHeader(http.StatusCreated)
		}
		if uerr := chefUser.UpdateFromJSON(userData); uerr != nil {
			jsonErrorReport(w, r, uerr.Error(), uerr.Status())
			return
		}

		if pk, pkfound := userData["public_key"]; pkfound {
			switch pk := pk.(type) {
			case string:
				if pkok, pkerr := user.ValidatePublicKey(pk); !pkok {
					jsonErrorReport(w, r, pkerr.Error(), http.StatusBadRequest)
					return
				}
				chefUser.SetPublicKey(pk)
				jsonUser["public_key"] = pk
			case nil:
				//show_public_key = false

			default:
				jsonErrorReport(w, r, "Bad request", http.StatusBadRequest)
				return
			}
		}

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
		if lerr := loginfo.LogEvent(opUser, chefUser, "modify"); lerr != nil {
			jsonErrorReport(w, r, lerr.Error(), http.StatusInternalServerError)
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
