/* Utililty functions and methods. Should probably absorbe what's in "common.go"
 * right now. */

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

/*
Package util contains various utility functions that are useful across all of goiardi.
*/
package util

import (
	"fmt"
	"github.com/ctdk/goiardi/config"
	"net/http"
	"reflect"
	"strconv"
	"regexp"
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

// An error type that includes an http status code (defaults to 
// http.BadRequest).
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

// Create a new Gerror, with a formatted error string.
func Errorf(format string, a ...interface{}) Gerror {
	return New(fmt.Sprintf(format, a...))
}

// Returns the Gerror error message.
func (e *gerror) Error() string {
	return e.msg
}

// Set the Gerror HTTP status code.
func (e *gerror) SetStatus(s int) {
	e.status = s
}

// Returns the Gerror's HTTP status code.
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

// Flatten an object and expand its keys into a map[string]string so it's 
// suitable for indexing, either with solr (eventually) or with the whipped up
// replacement for local mode. Objects fed into this function *must* have the
// "json" tag set for their struct members.
func FlattenObj(obj interface{}) map[string]interface{} {
	expanded := make(map[string]interface{})
	s := reflect.ValueOf(obj).Elem()
	for i := 0; i < s.NumField(); i++ {
		v := s.Field(i).Interface()
		key := s.Type().Field(i).Tag.Get("json")
		var mergeKey string
		if key == "automatic" || key == "normal" || key == "default" || key == "override" || key == "raw_data" {
			mergeKey = ""
		} else {
			mergeKey = key
		}
		subExpand := DeepMerge(mergeKey, v)
		/* Now merge the returned map */
		for k, u := range subExpand {
			expanded[k] = u
		}
	}
	return expanded
}

// Turn an object into a map[string]interface{}. Useful for when you have a
// slice of objects that you need to trim, mutilate, fold, etc. before returning
// them as JSON.
func MapifyObject(obj interface{}) map[string]interface{} {
	mapified := make(map[string]interface{})
	s := reflect.ValueOf(obj).Elem()
	for i := 0; i < s.NumField(); i++ {
		v := s.Field(i).Interface()
		key := s.Type().Field(i).Tag.Get("json")
		mapified[key] = v
	}
	return mapified
}

// Given a flattened object, prepares it for indexing by turning it into a 
// sorted slice of strings formatted like "key:value".
func Indexify(flattened map[string]interface{}) []string {
	readyToIndex := make([]string, 0)
	for k, v := range flattened {
		switch v := v.(type) {
			case string:
				v = escapeStr(v)
				line := fmt.Sprintf("%s:%s", k, v)
				readyToIndex = append(readyToIndex, line)
			case []string:
				for _, w := range v {
					w = escapeStr(w)
					line := fmt.Sprintf("%s:%s", k, w)
					readyToIndex = append(readyToIndex, line)
				}
			default:
				err := fmt.Errorf("We should never have been able to reach this state. Key %s had a value %v of type %T", k, v, v)
				panic(err)
		}
	}
	sort.Strings(readyToIndex)
	return readyToIndex
}

func escapeStr(s string) string {
	s = strings.Replace(s, "[", "\\[", -1)
	s = strings.Replace(s, "]", "\\]", -1)
	s = strings.Replace(s, "::", "\\:\\:", -1)
	return s
}

// Merge disparate data structures into a flat hash.
func DeepMerge(key string, source interface{}) map[string]interface{} {
	merger := make(map[string]interface{})
	switch v := source.(type) {
		case map[string]interface{}:
			/* We also need to get things like 
			 * "default_attributes:key" indexed. */
			topLev := make([]string, len(v))
			n := 0
			for k, u := range v {
				if key != "" {
					topLev[n] = k
					n++
				}
				var nkey string
				if key == "" {
					nkey = k
				} else {
					nkey = fmt.Sprintf("%s_%s", key, k)
				}
				nm := DeepMerge(nkey, u)
				for j, q := range nm {
					merger[j] = q
				}
			}
			if key != "" {
				merger[key] = topLev
			}
		case map[string]string:
			/* We also need to get things like 
			 * "default_attributes:key" indexed. */
			topLev := make([]string, len(v))
			n := 0
			for k, u := range v {
				if key != "" {
					topLev[n] = k
					n++
				}
				var nkey string
				if key == "" {
					nkey = k
				} else {
					nkey = fmt.Sprintf("%s_%s", key, k)
				}
				merger[nkey] = u
			}
			if key != "" {
				merger[key] = topLev
			}

		case []interface{}:
			km := make([]string, len(v))
			for i, w := range v {
				km[i] = stringify(w)
			}
			merger[key] = km
		case []string:
			km := make([]string, len(v))
			for i, w := range v {
				km[i] = stringify(w)
			}
			merger[key] = km
			/* If this is the run list, break recipes and roles out
			 * into their own separate indexes as well. */
			if key == "run_list" {
				roleMatch := regexp.MustCompile(`^(recipe|role)\[(.*)\]`)
				roles := make([]string, 0)
				recipes := make([]string, 0)
				for _, w := range v {
					rItem := roleMatch.FindStringSubmatch(stringify(w))
					if rItem != nil {
						rType := rItem[1]
						rThing := rItem[2]
						if rType == "role" {
							roles = append(roles, rThing)
						} else if rType == "recipe" {
							recipes = append(recipes, rThing)
						}
					} 
				}
				if len(roles) > 0 {
					merger["role"] = roles
				}
				if len(recipes) > 0 {
					merger["recipe"] = recipes
				}
			} 
		default:
			merger[key] = stringify(v)
	}
	return merger
}

func stringify(source interface{}) string{
	switch s := source.(type) {
		case string:
			return s
		case uint8, uint16, uint32, uint64:
			n := reflect.ValueOf(s).Uint()
			str := strconv.FormatUint(n, 10)
			return str
		case int8, int16, int32, int64:
			n := reflect.ValueOf(s).Int()
			str := strconv.FormatInt(n, 10)
			return str
		case float32, float64:
			n := reflect.ValueOf(s).Float()
			str := strconv.FormatFloat(n, 'f', -1, 64)
			return str
		case bool:
			str := strconv.FormatBool(s)
			return str
		default:
			/* Just send back whatever %v gives */
			str := fmt.Sprintf("%v", s)
			return str
	}
}
