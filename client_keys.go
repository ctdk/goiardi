/*
 * Copyright (c) 2013-2026, Jeremy Bingham (<jeremy@goiardi.gl>)
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

// Client key handlers

package main

import (
	"encoding/json"
	"github.com/ctdk/goiardi/client"
	"github.com/ctdk/goiardi/logger"
	"github.com/ctdk/goiardi/reqctx"
	"github.com/ctdk/goiardi/util"
	"github.com/gorilla/mux"
	"net/http"
)

func clientKeysHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)
	org, orgerr := reqctx.CtxOrg(r.Context())
	if orgerr != nil {
		jsonErrorReport(w, r, orgerr.Error(), orgerr.Status())
		return
	}

	clientName := vars["name"]
	opUser, oerr := reqctx.CtxReqUser(r.Context())
	if oerr != nil {
		jsonErrorReport(w, r, oerr.Error(), oerr.Status())
		return
	}

	switch r.Method {
	case http.MethodGet:
		chefClient, gerr := client.Get(org, clientName)

		if gerr != nil {
			jsonErrorReport(w, r, gerr.Error(), gerr.Status())
			return
		}

		if f, err := org.PermCheck.CheckItemPerm(chefClient, opUser, "read"); err != nil {
			jsonErrorReport(w, r, err.Error(), err.Status())
			return
		} else if !f && (opUser.IsValidator() || !opUser.IsSelf(chefClient)) {
			jsonErrorReport(w, r, "You are not allowed to perform that action.", http.StatusForbidden)
			return
		}

		keyInfo := chefClient.GetKeyInfo()

		enc := json.NewEncoder(w)
		if err := enc.Encode(&keyInfo); err != nil {
			jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
			return
		}
	case http.MethodPost:
		chefClient, gerr := client.Get(org, clientName)

		if gerr != nil {
			jsonErrorReport(w, r, gerr.Error(), gerr.Status())
			return
		}
		keyData, jerr := parseObjJSON(r.Body)
		if jerr != nil {
			logger.Debugf("couldn't parse JSON POST body for client key creation: %s", jerr.Error())
			jsonErrorReport(w, r, jerr.Error(), http.StatusBadRequest)
			return
		}
		if f, err := org.PermCheck.CheckItemPerm(chefClient, opUser, "update"); err != nil {
			jsonErrorReport(w, r, err.Error(), err.Status())
			return
		} else if !f && (opUser.IsValidator() || !opUser.IsSelf(chefClient)) {
			jsonErrorReport(w, r, "You are not allowed to perform that action.", http.StatusForbidden)
			return
		}

		// try to make the key
		k, kerr := client.KeyFromJSON(keyData)
		if kerr != nil {
			jsonErrorReport(w, r, kerr.Error(), kerr.Status())
			return
		}
		if kerr = chefClient.SetNamedKey(k); kerr != nil {
			jsonErrorReport(w, r, kerr.Error(), kerr.Status())
			return
		}
		// if it worked, make the response
		resp := make(map[string]string)
		resp["uri"] = util.CustomObjURL(chefClient, util.JoinStr("/keys/", k.Name))
		enc := json.NewEncoder(w)
		if err := enc.Encode(&resp); err != nil {
			jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
			return
		}
	default:
		jsonErrorReport(w, r, "Unrecognized method for client keys!", http.StatusMethodNotAllowed)
	}
}

func clientIndividualKeyHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)
	org, orgerr := reqctx.CtxOrg(r.Context())
	if orgerr != nil {
		jsonErrorReport(w, r, orgerr.Error(), orgerr.Status())
		return
	}

	clientName := vars["name"]
	keyName := vars["key"]
	opUser, oerr := reqctx.CtxReqUser(r.Context())
	if oerr != nil {
		jsonErrorReport(w, r, oerr.Error(), oerr.Status())
		return
	}

	var perm string
	switch r.Method {
	case http.MethodGet:
		perm = "read"
	case http.MethodPut:
		perm = "update"
	case http.MethodDelete:
		perm = "delete"
	default:
		jsonErrorReport(w, r, "Unrecognized method for individual client key!", http.StatusMethodNotAllowed)
		return
	}

	chefClient, gerr := client.Get(org, clientName)
	if gerr != nil {
		jsonErrorReport(w, r, gerr.Error(), gerr.Status())
		return
	}

	if f, err := org.PermCheck.CheckItemPerm(chefClient, opUser, perm); err != nil {
		jsonErrorReport(w, r, err.Error(), err.Status())
		return
	} else if !f && (opUser.IsValidator() || !opUser.IsSelf(chefClient)) {
		jsonErrorReport(w, r, "You are not allowed to perform that action.", http.StatusForbidden)
		return
	}

	key := chefClient.NamedPublicKey(keyName)
	if key == nil {
		jsonErrorReport(w, r, "key not found", http.StatusNotFound)
		return
	}

	switch r.Method {
	case http.MethodGet:
		; // don't actually need to do anything else, but we don't want
		  // an error either
	case http.MethodPut:
		// can keys be renamed? I'm unsure
		keyData, jerr := parseObjJSON(r.Body)
		if jerr != nil {
			logger.Debugf("couldn't parse JSON POST body for client key update: %s", jerr.Error())
			jsonErrorReport(w, r, jerr.Error(), http.StatusBadRequest)
			return
		}

		nk, kerr := client.KeyFromJSON(keyData)
		if kerr != nil {
			jsonErrorReport(w, r, kerr.Error(), kerr.Status())
			return
		}
		// err right now if key names don't match
		if key.Name != nk.Name {
			jsonErrorReport(w, r, "key names don't match", http.StatusBadRequest)
			return
		}
		kerr = chefClient.SetNamedKey(nk)
		// reload key to be safe instead of passing nk back
		key = chefClient.NamedPublicKey(keyName)
	case http.MethodDelete:
		if kerr := chefClient.DeleteKey(keyName); kerr != nil {
			jsonErrorReport(w, r, kerr.Error(), kerr.Status())
			return
		}
	default:
		jsonErrorReport(w, r, "Unrecognized method for individual client key!", http.StatusMethodNotAllowed)
	}

	enc := json.NewEncoder(w)
	if err := enc.Encode(&key); err != nil {
		jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
		return
	}
}
