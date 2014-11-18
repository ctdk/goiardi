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
	"encoding/json"
	"fmt"
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/user"
	"net/http"
	"log"
)

type authenticator struct {
	Name, Password string
}
type authResponse struct {
	User     map[string]interface{} `json:"user"`
	Status string   `json:"status"`
}

func authenticateUserHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != "POST" {
		jsonErrorReport(w, r, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	dec := json.NewDecoder(r.Body)
	authJSON := make(map[string]interface{})
	if err := dec.Decode(&authJSON); err != nil {
		jsonErrorReport(w, r, err.Error(), http.StatusBadRequest)
		return
	}
	log.Printf("authJSON now: %+v", authJSON)
	auth, authErr := validateJSON(authJSON)
	if authErr != nil {
		jsonErrorReport(w, r, authErr.Error(), http.StatusBadRequest)
		return
	}

	resp := validateLogin(auth)

	enc := json.NewEncoder(w)
	if err := enc.Encode(resp); err != nil {
		jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
	}
}

func validateLogin(auth *authenticator) authResponse {
	// Check passwords and such later.
	// Automatically validate if UseAuth is not on
	var resp authResponse
	
	u, err := user.Get(auth.Name)
	if err != nil {
		resp.Status = "unknown"
		return resp
	}
	resp.User = u.ToJSON()
	delete(resp.User, "public_key")

	if !config.Config.UseAuth {
		resp.Status = "linked"
		return resp
	}
	perr := u.CheckPasswd(auth.Password)
	if perr != nil {
		resp.Status = "failed"
	} else {
		// TODO: Check association with this org, I believe
		resp.Status = "linked"
	}
	return resp
}

func validateJSON(authJSON map[string]interface{}) (*authenticator, error) {
	auth := new(authenticator)
	if name, ok := authJSON["name"]; ok {
		switch name := name.(type) {
		case string:
			auth.Name = name
		default:
			err := fmt.Errorf("Field 'name' invalid")
			return nil, err
		}
	} else if name, ok := authJSON["username"]; ok {
		switch name := name.(type) {
		case string:
			auth.Name = name
		default:
			err := fmt.Errorf("Field 'username' invalid")
			return nil, err
		}
	} else {
		err := fmt.Errorf("Field 'username' missing")
		return nil, err
	}
	if auth.Name == "" {
		err := fmt.Errorf("Field 'username' missing")
		return nil, err
	}
	if passwd, ok := authJSON["password"]; ok {
		switch passwd := passwd.(type) {
		case string:
			if passwd == "" {
				err := fmt.Errorf("Field 'password' invalid")
				return nil, err
			}
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
