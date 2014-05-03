/* Some common definitions, interfaces, etc. */

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
	"io"
	"encoding/json"
	"net/http"
	"git.tideland.biz/goas/logger"
	"strings"
	"fmt"
)

func ParseObjJson(data io.ReadCloser) (map[string]interface{}, error){
	obj_data := make(map[string]interface{})
	dec := json.NewDecoder(data)
	
	if err := dec.Decode(&obj_data); err != nil {
		return nil, err
	}

	/* If this kind of object comes with a run_list, process it */
	if _, ok := obj_data["run_list"]; ok {
		if rl, err := chkRunList(obj_data["run_list"]); err != nil {
			return nil, err
		} else {
			obj_data["run_list"] = rl
		}
	}

	/* And if we have env_run_lists */
	if _, ok := obj_data["env_run_lists"]; ok {
		switch erl := obj_data["env_run_lists"].(type) {
			case map[string]interface{}:
				new_env_run_list := make(map[string][]string, len(erl))
				var erlerr error
				for i, v := range erl {
					if new_env_run_list[i], erlerr = chkRunList(v); erlerr != nil {
						erlerr := fmt.Errorf("Field 'env_run_lists' contains invalid run lists")
						return nil, erlerr
					}
				}
				obj_data["env_run_lists"] = new_env_run_list
			default:
				err := fmt.Errorf("Field 'env_run_lists' contains invalid run lists")
				return nil, err
		}
	}

	/* If this kind of object has any attributes, process them too */
	attributes := []string{ "normal", "default", "automatic", "override", "default_attributes", "override_attributes" }
	for _, k := range attributes {
		/* Don't add if it doesn't exist in the json data at all */
		if _, ok := obj_data[k]; ok {
			if obj_data[k] == nil {
				obj_data[k] = make(map[string]interface{})
			}
		}
	}

	return obj_data, nil
}

func SplitPath(path string) (split_path []string){
	split_path = strings.Split(path[1:], "/")
	return split_path
}

func JsonErrorReport(w http.ResponseWriter, r *http.Request, error_str string, status int){
	logger.Infof(error_str)
	json_error := map[string][]string{ "error": []string{ error_str } }
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	if err:= enc.Encode(&json_error); err != nil {
		logger.Errorf(err.Error())
	}
	return
}

func CheckAccept(w http.ResponseWriter, r *http.Request, acceptType string) error {
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
	switch o := rl.(type){
		case []interface{}:
			_ = o
			new_run_list := make([]string, len(o))
			for i, v := range o {
				switch v := v.(type) {
					case string:
						new_run_list[i] = v
					default:
						err := fmt.Errorf("Field 'run_list' is not a valid run list")
						return nil, err
				}
			}
			return new_run_list, nil
		default:
			err := fmt.Errorf("Field 'run_list' is not a valid run list")
			return nil, err
	}
}
