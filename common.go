/* Some common definitions, interfaces, etc. */

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
	"github.com/ctdk/goiardi/util"
	"io"
	"net/http"
	"strings"
)

func parseObjJSON(data io.ReadCloser) (map[string]interface{}, error) {
	objData := make(map[string]interface{})
	dec := json.NewDecoder(data)
	dec.UseNumber()

	if err := dec.Decode(&objData); err != nil {
		return nil, err
	}
	return checkAttrs(objData)
}

func checkAttrs(objData map[string]interface{}) (map[string]interface{}, error) {
	/* If this kind of object comes with a run_list, process it */
	if _, ok := objData["run_list"]; ok {
		rl, err := chkRunList(objData["run_list"])
		if err != nil {
			return nil, err
		}
		objData["run_list"] = rl
	}

	/* And if we have env_run_lists */
	if _, ok := objData["env_run_lists"]; ok {
		switch erl := objData["env_run_lists"].(type) {
		case map[string]interface{}:
			newEnvRunList := make(map[string][]string, len(erl))
			var erlerr error
			for i, v := range erl {
				if newEnvRunList[i], erlerr = chkRunList(v); erlerr != nil {
					erlerr := fmt.Errorf("Field 'env_run_lists' contains invalid run lists")
					return nil, erlerr
				}
			}
			objData["env_run_lists"] = newEnvRunList
		default:
			err := fmt.Errorf("Field 'env_run_lists' contains invalid run lists")
			return nil, err
		}
	}

	/* If this kind of object has any attributes, process them too */
	attributes := []string{"normal", "default", "automatic", "override", "default_attributes", "override_attributes"}
	for _, k := range attributes {
		/* Don't add if it doesn't exist in the json data at all */
		if _, ok := objData[k]; ok {
			if objData[k] == nil {
				objData[k] = make(map[string]interface{})
			}
		}
	}

	return objData, nil
}

func splitPath(path string) []string {
	sp := strings.Split(path[1:], "/")
	return sp
}

func jsonErrorReport(w http.ResponseWriter, r *http.Request, errorStr string, status int) {
	util.JSONErrorReport(w, r, errorStr, status)
	return
}

func jsonErrorNonArrayReport(w http.ResponseWriter, r *http.Request, errorStr string, status int) {
	util.JSONErrorNonArrayReport(w, r, errorStr, status)
	return
}

func checkAccept(w http.ResponseWriter, r *http.Request, acceptType string) error {
	for _, at := range r.Header["Accept"] {
		if at == "*/*" {
			return nil // we accept all types in this case
		} else if at == acceptType {
			return nil
		}
	}
	err := fmt.Errorf("Client cannot accept content type %s", acceptType)
	return err
}

func chkRunList(rl interface{}) ([]string, error) {
	switch o := rl.(type) {
	case []interface{}:
		newRunList := make([]string, len(o))
		for i, v := range o {
			switch v := v.(type) {
			case string:
				newRunList[i] = v
			default:
				err := fmt.Errorf("Field 'run_list' is not a valid run list")
				return nil, err
			}
		}
		return newRunList, nil
	default:
		err := fmt.Errorf("Field 'run_list' is not a valid run list")
		return nil, err
	}
}
