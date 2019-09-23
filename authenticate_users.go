/* Authenticate_user functions */

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
	"github.com/ctdk/goiardi/actor"
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/user"
	"github.com/ctdk/goiardi/util"
	"net/http"
)

type authenticator struct {
	Name, Password string
}
type authResponse struct {
	User   map[string]interface{} `json:"user"`
	Status string                 `json:"status"`
}

func authenticateUserHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != "POST" {
		jsonErrorReport(w, r, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	opUser, oerr := actor.GetReqUser(nil, r.Header.Get("X-OPS-USERID"))

	dec := json.NewDecoder(r.Body)
	dec.UseNumber()
	authJSON := make(map[string]interface{})
	if err := dec.Decode(&authJSON); err != nil {
		jsonErrorReport(w, r, err.Error(), http.StatusBadRequest)
		return
	}

	auth, authErr := validateJSON(authJSON)
	if authErr != nil {
		jsonErrorReport(w, r, authErr.Error(), authErr.Status())
		return
	}

	resp, rerr := validateLogin(auth)
	if rerr != nil {
		s := rerr.Status()
		// Another area that I can
		if !opUser.IsAdmin() {
			s = http.StatusForbidden
		}
		jsonErrorReport(w, r, rerr.Error(), s)
		return
	}
	// seems like this ought to be one of the first things done, but here
	// we are.
	if oerr != nil {
		jsonErrorReport(w, r, oerr.Error(), oerr.Status())
		return
	}
	if !opUser.IsAdmin() {
		jsonErrorReport(w, r, "not permitted for this user", http.StatusForbidden)
		return
	}

	enc := json.NewEncoder(w)
	if err := enc.Encode(resp); err != nil {
		jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
	}
}

func validateLogin(auth *authenticator) (authResponse, util.Gerror) {
	// Check passwords and such later.
	// Automatically validate if UseAuth is not on
	var resp authResponse

	u, err := user.Get(auth.Name)
	if err != nil {
		gerr := util.Errorf("Failed to authenticate: Username and password incorrect")
		gerr.SetStatus(http.StatusUnauthorized)
		return resp, gerr
	}
	// cannot allow the superuser to actually log in this way
	if u.IsAdmin() {
		gerr := util.Errorf("forbidden")
		gerr.SetStatus(http.StatusForbidden)
		return resp, gerr
	}
	resp.User = u.ToJSON()
	delete(resp.User, "public_key")

	if !config.Config.UseAuth {
		resp.Status = "linked"
		return resp, nil
	}
	perr := u.CheckPasswd(auth.Password)
	if perr != nil {
		gerr := util.CastErr(perr)
		gerr.SetStatus(http.StatusUnauthorized)
		return resp, gerr
	} else {
		// TODO: Check association with this org, I believe
		resp.Status = "linked"
	}
	return resp, nil
}

func validateJSON(authJSON map[string]interface{}) (*authenticator, util.Gerror) {
	auth := new(authenticator)
	/* for k := range authJSON {
		if k != "name" && k != "username" && k != "password" {
			err := util.Errorf("invalid key %s", k)
			err.SetStatus(http.StatusForbidden)
			return auth, err
		}
	} */
	if name, ok := authJSON["name"]; ok {
		switch name := name.(type) {
		case string:
			auth.Name = name
		default:
			err := util.Errorf("Field 'name' invalid")
			return nil, err
		}
	} else if name, ok := authJSON["username"]; ok {
		switch name := name.(type) {
		case string:
			auth.Name = name
		default:
			err := util.Errorf("Field 'username' invalid")
			return nil, err
		}
	} else {
		err := util.Errorf("Field 'username' missing")
		return nil, err
	}
	if auth.Name == "" {
		err := util.Errorf("Field 'username' missing")
		return nil, err
	}
	if passwd, ok := authJSON["password"]; ok {
		switch passwd := passwd.(type) {
		case string:
			if passwd == "" {
				err := util.Errorf("Field 'password' invalid")
				return nil, err
			}
			auth.Password = passwd
		default:
			err := util.Errorf("Field 'password' invalid")
			return nil, err
		}
	} else {
		err := util.Errorf("Field 'password' missing")
		return nil, err
	}
	return auth, nil
}
