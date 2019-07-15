/* Utility functions and methods. Should probably absorbe what's in "common.go"
 * right now. */

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

/*
Package util contains various utility functions that are useful across all of goiardi.
*/
package util

import (
	"encoding/json"
	"fmt"
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/gerror"
	"github.com/pborman/uuid"
	"github.com/tideland/golib/logger"
	"net/http"
	"reflect"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
)

// hopefully a reasonable starting map allocation for DeepMerge if the type
// isn't a map
const defaultMapCap = 4

// SearchSchemaSkel is a printf style format string used to generate the
// expected search schema for a given organization. Moved out of organizations
// so it can be easily referred to inside the indexer.
const SearchSchemaSkel = "goiardi_search_org_%d"

// BaseSearchSchema is a constant holding the name of the base search schema in
// goiardi. This schema is cloned for each organization.
const BaseSearchSchema = "goiardi_search_base"

// declare some postgres search key regexps once up here, so they aren't
// reallocated every time the function is called.

var re *regexp.Regexp
var reQuery *regexp.Regexp
var bs *regexp.Regexp
var ps *regexp.Regexp

// And a regexp for matching roles in DeepMerge
var roleMatch *regexp.Regexp

func init() {
	re = regexp.MustCompile(`[^\pL\pN_\.]`)
	reQuery = regexp.MustCompile(`[^\pL\pN_\.\*\?]`)
	bs = regexp.MustCompile(`_{2,}`)
	ps = regexp.MustCompile(`\.{2,}`) // repeated . will cause trouble too
	roleMatch = regexp.MustCompile(`^(recipe|role)\[(.*)\]`)
}

// NoDBConfigured is an error for when no database has been configured for use,
// yet an SQL function is being called.
var NoDBConfigured = gerror.StatusError("no db configured, but you tried to use one", http.StatusInternalServerError)

// GoiardiObj is an interface for helping goiardi/chef objects, like cookbooks,
// roles, etc., be able to easily make URLs and be identified by name.
type GoiardiObj interface {
	GetName() string
	URLType() string
	OrgName() string
}

// Gerror is an error type that wraps around the goiardi Error type.
type Gerror interface {
	gerror.Error
}

// Errorf creates a new Gerror, with a formatted error string. A convenience
// wrapper around error.Errorf.
func Errorf(format string, a ...interface{}) Gerror {
	return gerror.Errorf(format, a...)
}

// CastErr will easily cast a different kind of error to a Gerror. A convenience
// wrapper around error.CastErr.
func CastErr(err error) Gerror {
	return gerror.CastErr(err)
}

// ObjURL crafts a URL for an object.
func ObjURL(obj GoiardiObj) string {
	baseURL := config.ServerBaseURL()

	fullURL := fmt.Sprintf("%s/organizations/%s/%s/%s", baseURL, obj.OrgName(), obj.URLType(), obj.GetName())
	return fullURL
}

// BaseObjURL crafts a URL for an object outside of an organization.
func BaseObjURL(obj GoiardiObj) string {
	baseURL := config.ServerBaseURL()

	fullURL := fmt.Sprintf("%s/%s/%s", baseURL, obj.URLType(), obj.GetName())
	return fullURL
}

// CustomObjURL crafts a URL for a Goiardi object with additional path elements.
func CustomObjURL(obj GoiardiObj, path string) string {
	chkPath(&path)
	return fmt.Sprintf("%s%s", ObjURL(obj), path)
}

// CustomURL crafts a URL from the provided path, without providing an object.
func CustomURL(path string) string {
	chkPath(&path)
	return fmt.Sprintf("%s%s", config.ServerBaseURL(), path)
}

func chkPath(p *string) {
	if (*p)[0] != '/' {
		*p = fmt.Sprintf("/%s", *p)
	}
}

// FlattenObj flattens an object and expand its keys into a map[string]string so
// it's suitable for indexing, either with solr (eventually) or with the whipped
// up replacement for local mode. Objects fed into this function *must* have the
// "json" tag set for their struct members.
func FlattenObj(obj interface{}) map[string]interface{} {
	s := reflect.ValueOf(obj).Elem()
	expanded := make(map[string]interface{}, s.NumField())

	for i := 0; i < s.NumField(); i++ {
		if !s.Field(i).CanInterface() {
			continue
		}
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

// MapifyObject turns an object into a map[string]interface{}. Useful for when
// you have a slice of objects that you need to trim, mutilate, fold, etc.
// before returning them as JSON.
func MapifyObject(obj interface{}) map[string]interface{} {
	mapified := make(map[string]interface{})
	s := reflect.ValueOf(obj).Elem()
	for i := 0; i < s.NumField(); i++ {
		if !s.Field(i).CanInterface() {
			continue
		}
		v := s.Field(i).Interface()
		key := s.Type().Field(i).Tag.Get("json")
		mapified[key] = v
	}
	return mapified
}

// Indexify prepares a flattened object for indexing by turning it into a sorted
// slice of strings formatted like "key:value".
func Indexify(flattened map[string]interface{}) []string {
	// rough and ready allocation - allocate for the number of elements in
	// the flattened map. It's likely to be more, but it's a good start.
	// If it's too inaccurate it may be worth addressing however.
	readyToIndex := make([]string, 0, len(flattened))

	// keep values in the index down to a reasonable size
	maxValLen := config.Config.IndexValTrim

	for k, v := range flattened {
		switch v := v.(type) {
		case string:
			//v = IndexEscapeStr(v)
			v = TrimStringMax(v, maxValLen)
			line := strings.Join([]string{k, v}, ":")
			readyToIndex = append(readyToIndex, line)
		case []string:
			sort.Strings(v)
			v = RemoveDupStrings(v)
			for _, w := range v {
				//w = IndexEscapeStr(w)
				w = TrimStringMax(w, maxValLen)
				line := strings.Join([]string{k, w}, ":")
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

// IndexEscapeStr escapes values to index in the database, so characters that
// need to be escaped for Solr are properly found when using the trie or
// postgres based searches.
func IndexEscapeStr(s string) string {
	s = strings.Replace(s, "[", "\\[", -1)
	s = strings.Replace(s, "]", "\\]", -1)
	s = strings.Replace(s, "::", "\\:\\:", -1)
	return s
}

// DeepMerge merges disparate data structures into a flat hash.
func DeepMerge(key string, source interface{}) map[string]interface{} {
	refIface := reflect.ValueOf(source)
	var mapCap int
	if refIface.Kind() == reflect.Map {
		mapCap = refIface.Len()
	} else {
		mapCap = defaultMapCap
	}

	merger := make(map[string]interface{}, mapCap)
	var sep string
	if config.Config.DotSearch {
		sep = "."
	} else {
		sep = "_"
	}
	switch v := source.(type) {
	case map[string]interface{}:
		/* We also need to get things like
		 * "default_attributes:key" indexed. */
		topLev := make([]string, len(v))
		n := 0
		for k, u := range v {
			if key != "" && !config.Config.UsePostgreSQL {
				topLev[n] = k
				n++
			}
			nkey := getNKey(key, k, sep)
			nm := DeepMerge(nkey, u)
			for j, q := range nm {
				merger[j] = q
			}
		}
		if key != "" && !config.Config.UsePostgreSQL {
			merger[key] = topLev
		}
	case map[string]string:
		/* We also need to get things like
		 * "default_attributes:key" indexed. */
		topLev := make([]string, len(v))
		n := 0
		for k, u := range v {
			if key != "" && !config.Config.UsePostgreSQL {
				topLev[n] = k
				n++
			}
			nkey := getNKey(key, k, sep)

			merger[nkey] = u
		}
		if key != "" && !config.Config.UsePostgreSQL {
			merger[key] = topLev
		}

	case []interface{}:
		km := make([]string, 0, len(v))
		mapMerge := make(map[string][]string)
		for _, w := range v {
			// If it's an array of maps or arrays, deep merge them
			// properly. Otherwise, stringify as best we can.
			vRef := reflect.ValueOf(w)
			if vRef.Kind() == reflect.Map {
				interMap := DeepMerge("", w)
				for imk, imv := range interMap {
					nk := getNKey(key, imk, sep)
					// Anything that's come back from
					// DeepMerge should be a string.
					mapMerge[nk] = mergeInterfaceMapChildren(mapMerge[nk], imv)
				}
			} else if vRef.Kind() == reflect.Slice {
				for _, sv := range w.([]interface{}) {
					smMerge := DeepMerge("", sv)
					// WARNING: This *may* be a little iffy
					// still, there are some very weird
					// possibilities under this that need
					// more testing.
					for smk, smv := range smMerge {
						if smk == "" {
							km = mergeInterfaceMapChildren(km, smv)
						} else {
							nk := getNKey(key, smk, sep)
							mapMerge[nk] = mergeInterfaceMapChildren(mapMerge[nk], smv)
						}
					}
				}
			} else {
				s := stringify(w)
				km = append(km, s)
			}
		}
		for mmi, mmv := range mapMerge {
			merger[mmi] = mmv
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
			roles := make([]string, 0, len(v))
			recipes := make([]string, 0, len(v))
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

func getNKey(key string, subkey string, sep string) string {
	var nkey string
	if key == "" {
		nkey = subkey
	} else {
		nkey = strings.Join([]string{key, subkey}, sep)
	}
	return nkey
}

func mergeInterfaceMapChildren(strArr []string, val interface{}) []string {
	if reflect.ValueOf(val).Kind() == reflect.Slice {
		strArr = append(strArr, val.([]string)...)
	} else {
		strArr = append(strArr, val.(string))
	}
	return strArr
}

func stringify(source interface{}) string {
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

// JoinStr joins strings together. It's just a wrapper around strings.Join at
// the moment, but could be replaced by another method should something else
// pop up.
func JoinStr(str ...string) string {
	return strings.Join(str, "")
}

// JSONErrorReport spits out the given error and HTTP status code formatted as
// JSON for the client's benefit before completing the errored out request.
func JSONErrorReport(w http.ResponseWriter, r *http.Request, errorStr string, status int) {
	spewCallers()
	logger.Infof(errorStr)
	jsonError := map[string][]string{"error": []string{errorStr}}
	sendErrorReport(w, jsonError, status)
	return
}

func JSONErrorNonArrayReport(w http.ResponseWriter, r *http.Request, errorStr string, status int) {
	spewCallers()
	logger.Infof(errorStr)
	jsonError := map[string]string{"error": errorStr}
	sendErrorReport(w, jsonError, status)
	return
}

func JSONErrorMapReport(w http.ResponseWriter, r *http.Request, errMap map[string]interface{}, status int) {
	spewCallers()
	logger.Infof("%+v", errMap)
	sendErrorReport(w, errMap, status)
	return
}

func sendErrorReport(w http.ResponseWriter, jsonError interface{}, status int) {
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	if err := enc.Encode(&jsonError); err != nil {
		logger.Errorf(err.Error())
	}
	return
}

func MakeAuthzID() string {
	return fmt.Sprintf("%32x", []byte(uuid.NewRandom()))
}

// PgSearchKey removes characters from search term fields that make the ltree
// data type unhappy. This leads to the postgres-based search being, perhaps,
// somewhat less precise than the solr (or ersatz solr) based search, but at the
// same time one that's less resource demanding and covers almost all known use
// cases. Potential bug: Postgres considers some, but not all, unicode letters
// as being alphanumeric; i.e. golang and postgres both consider 'ü' to be a
// letter, but golang accepts 'ሀ' as a letter while postgres does not. This is
// reasonably unlikely to be an issue, but if you're using lots of non-European
// characters in your attributes this could be a problem. We're accepting more
// than raw ASCII alnum however because it's better behavior and because
// Postgres does accept at least some other alphabets as being alphanumeric.
func PgSearchKey(key string) string {
	return pgKeyReplace(key, re, bs, ps)
}

// PgSearchQueryKey is very similar to PgSearchKey, except that it preserves the
// Solr wildcard charactes '*' and '?' in the queries.
func PgSearchQueryKey(key string) string {
	return pgKeyReplace(key, reQuery, bs, ps)
}

func pgKeyReplace(key string, re, bs, ps *regexp.Regexp) string {
	k := re.ReplaceAllString(key, "_")
	k = bs.ReplaceAllString(k, "_")
	k = ps.ReplaceAllString(k, ".")
	k = strings.Trim(k, "_")
	// on the off hand chance we get leading or trailing dots
	k = strings.Trim(k, ".")
	// finally, if converting search query syntax, convert all _ to '.'.
	// This may need to be revisited in more detail if we find ourselves
	// needing more finesse with escaping underscores.
	if config.Config.ConvertSearch {
		k = strings.Replace(k, "_", ".", -1)
		k = ps.ReplaceAllString(k, ".")
	}
	return k
}

func spewCallers() {
	return // TODO: make this settable with a flag. Deactivating for now.
	pc := make([]uintptr, 10)
	n := runtime.Callers(2, pc)
	if n == 0 {
		return
	}
	pc = pc[:n]
	frames := runtime.CallersFrames(pc)
	logger.Debugf("printing %d frames", n)

	for {
		frame, more := frames.Next()
		logger.Debugf("- more:%v | %s", more, frame.Function)
		if !more {
			break
		}
	}
}
