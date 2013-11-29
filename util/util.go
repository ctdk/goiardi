/* Utililty functions and methods. Should probably absorbe what's in "common.go"
 * right now. */

/*
 * Copyright (c) 2013, Jeremy Bingham (<jbingham@gmail.com>)
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

/*
Package util contains various utility functions that are useful across all of goiardi.
*/
package util

import (
	"fmt"
	"github.com/ctdk/goiardi/config"
	"regexp"
	"net/http"
	"sort"
	"strings"
)

// Anything that implements these functions is a goiardi/chef object, like a
// cookbook, role, etc., and will be able to use these common functions.
type GoiardiObj interface {
	GetName() string
	URLType() string
}

type gerror struct {
	msg string
	status int
}

type Gerror interface {
	Error() string
	Status() int
	SetStatus(int)
}

func New(text string) Gerror {
	return &gerror{msg: text, 
		status: http.StatusBadRequest, 
		}
}

func Errorf(format string, a ...interface{}) Gerror {
	return New(fmt.Sprintf(format, a...))
}

func (e *gerror) Error() string {
	return e.msg
}

func (e *gerror) SetStatus(s int) {
	e.status = s
}

func (e *gerror) Status() int {
	return e.status
}

// Craft a URL
func ObjURL(obj GoiardiObj) string {
	base_url := config.ServerBaseURL()
	full_url := fmt.Sprintf("%s/%s/%s", base_url, obj.URLType(), obj.GetName())
	return full_url
}

// Craft a URL for a Goiardi object with additional path elements
func CustomObjURL(obj GoiardiObj, path string) string {
	chkPath(&path)
	return fmt.Sprintf("%s%s", ObjURL(obj), path)
}

// Craft a URL from the provided path, without providing an object.
func CustomURL(path string) string {
	chkPath(&path)
	return fmt.Sprintf("%s%s", config.ServerBaseURL(), path)
}

func chkPath(p *string){
	if (*p)[0] != '/' {
		*p = fmt.Sprintf("/%s", *p)
	}
}

func ValidateName(name string) bool {
	m, _ := regexp.MatchString("[^A-Za-z0-9_.-]", name)
	return !m
}

func ValidateDBagName(name string) bool {
	m, _ := regexp.MatchString("[^A-Za-z0-9_.:-]", name)
	return !m
}

func ValidateEnvName(name string) bool {
	m, _ := regexp.MatchString("[^A-Za-z0-9_-]", name)
	return !m
}

func ValidateAsString(str interface{}) (string, Gerror) {
	switch str := str.(type) {
		case string:
			return str, nil
		default:
			err := Errorf("Field 'name' missing")
			return "", err
	}
}

func ValidateAsBool(b interface{}) (bool, Gerror){
	switch b := b.(type) {
		case bool:
			return b, nil
		default:
			err := Errorf("Invalid bool")
			return false, err
	}
}

func ValidateAsFieldString(str interface{}) (string, Gerror){
	switch str := str.(type) {
		case string:
			return str, nil
		case nil:
			err := Errorf("Field 'name' nil")
			return "", err
		default:
			err := Errorf("Field 'name' missing")
			return "", err
	}
}

func ValidateAttributes(key string, attrs interface{}) (map[string]interface{}, Gerror){
	switch attrs := attrs.(type) {
		case map[string]interface{}:
			return attrs, nil
		case nil:
			/* Separate to do more validations above */
			nil_attrs := make(map[string]interface{})
			return nil_attrs, nil
		default:
			err := Errorf("Field '%s' is not a hash", key)
			return nil, err
	}
}

func ValidateRunList(rl interface{}) ([]string, Gerror) {
	switch rl := rl.(type) {
		case []string:
			for i, r := range rl {
				if j, err := validateRLItem(r); err != nil {
					return nil, err
				} else {
					if j == "" {
						err := Errorf("Field 'run_list' is not a valid run list")
						return nil, err
					} 
					rl[i] = j
				}
			}

			/* Remove dupes */
			rl_hash := make(map[string]string, len(rl))
			for _, u := range rl {
				rl_hash[u] = u
			}
			rl = make([]string, len(rl_hash))
			z := 0
			for k, _ := range rl_hash {
				rl[z] = k
				z++
			}

			// TODO: needs a more accurate sort
			sort.Strings(rl)
			return rl, nil
		case nil:
			/* separate to do more validations above */
			nil_rl := make([]string, 0)
			return nil_rl, nil
		default:
			err := Errorf("Not a proper runlist []string")
			return nil, err
	}
}

func validateRLItem(item string) (string, Gerror){
	/* There's a few places this might be used. */
	err := Errorf("Field 'run_list' is not a valid run list")

	if item == "" {
		return "", err
	}

	/* first checks */
	valid_rl := regexp.MustCompile("[^A-Za-z0-9_\\[\\]@\\.:]")
	m := valid_rl.MatchString(item)

	if m {
		return "", err
	}

	inspectRegexp := regexp.MustCompile(`^(\w+)\[(.*?)\]$`)
	inspect_item := inspectRegexp.FindStringSubmatch(item)

	if inspect_item != nil {
		rl_type := inspect_item[1]
		rl_item := inspect_item[2]
		if rl_type == "role" {
			if !validateRoleName(rl_item){
				return "", err
			}
		} else if rl_type == "recipe" {
			if !validateRecipeName(rl_item){
				return "", err
			}
		} else {
			return "", err
		}
	} else {
		if validateRecipeName(item) {
			item = fmt.Sprintf("recipe[%s]", item)
		} else {
			return "", err
		}
	}
	
	return item, nil
}

func validateRoleName(name string) bool {
	valid_role := regexp.MustCompile("[^A-Za-z0-9_-]")
	m := valid_role.MatchString(name)
	return !m
}

func validateRecipeName(name string) bool {
	first_valid := regexp.MustCompile("[^A-Za-z0-9_@\\.:]")
	m := first_valid.MatchString(name)
	if m {
		return false
	}

	/* If we have a version */
	if strings.Index(name, "@") != -1 {
		h := strings.Split(name, "@")
		name = h[0]
		version := h[1]
		valid_ver := regexp.MustCompile(`^\d\.\d(\.\d)?$`)
		if !valid_ver.MatchString(version){
			return false
		}
	}
	/* If we get this far, just do a final check on the name */
	final_chk := regexp.MustCompile(`^\w+(::\w+)?$`)
	n := final_chk.MatchString(name)

	return n
}
