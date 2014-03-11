/* Authenticate_user functions */

/*
 * Copyright (c) 2013-2014, Jeremy Bingham (<jbingham@gmail.com>)
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
	"net/http"
	"encoding/json"
	"fmt"
	"github.com/ctdk/goiardi/actor"
	"github.com/ctdk/goiardi/config"
)

type authenticator struct {
	Name, Password string
}
type authResponse struct {
	Name string `json:"name"`
	Verified bool `json:"verified"`
}

func authenticate_user_handler(w http.ResponseWriter, r *http.Request){
	/* Suss out what methods to allow */

	dec := json.NewDecoder(r.Body)
	authJson := make(map[string]interface{})
	if err := dec.Decode(&authJson); err != nil {
		JsonErrorReport(w, r, err.Error(), http.StatusBadRequest)
		return
	}
	auth, authErr := validateJson(authJson)
	if authErr != nil {
		JsonErrorReport(w, r, authErr.Error(), http.StatusBadRequest)
		return
	}

	resp := validateLogin(auth)

	enc := json.NewEncoder(w)
	if err := enc.Encode(resp); err != nil {
		JsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
	}
}

func validateLogin(auth *authenticator) authResponse {
	// Check passwords and such later.
	// Automatically validate if UseAuth is not on
	var resp authResponse
	resp.Name = auth.Name
	if !config.Config.UseAuth {
		resp.Verified = true
		return resp
	}
	u, err := actor.Get(auth.Name)
	if err != nil {
		resp.Verified = false
		return resp
	} 
	perr := u.CheckPasswd(auth.Password)
	if perr != nil {
		resp.Verified = false
	} else {
		resp.Verified = true
	}
	return resp
}

func validateJson(authJson map[string]interface{}) (*authenticator, error) {
	auth := new(authenticator)
	if name, ok := authJson["name"]; ok {
		switch name := name.(type) {
			case string:
				auth.Name = name
			default:
				err := fmt.Errorf("Field 'name' invalid")
				return nil, err
		}
	} else {
		err := fmt.Errorf("Field 'name' missing")
		return nil, err
	}
	if passwd, ok := authJson["password"]; ok {
		switch passwd := passwd.(type) {
			case string:
				auth.Password = passwd
			default:
				err := fmt.Errorf("Field 'password' invalid")
				return nil, err
		}
	} else {
		err := fmt.Errorf("Field 'password' missing")
		return nil, err
	}
	return auth, nil
}
