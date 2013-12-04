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
	"strconv"
	"github.com/ctdk/goiardi/filestore"
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
		case nil:
			err := Errorf("Field 'name' missing")
			return "", err
		default:
			err := Errorf("Field 'name' invalid")
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

func ValidateAsVersion(ver interface{}) (string, Gerror){
	switch ver := ver.(type) {
		case string:
			valid_ver := regexp.MustCompile(`^(\d+)\.(\d+)(\.?)(\d+)?$`)
			inspect_ver := valid_ver.FindStringSubmatch(ver)

			if inspect_ver != nil {
				nums := []int{ 1, 2, 4 }
				for _, n := range nums {
					/* #4 might not exist, but 1 and 2 must.
					 * The regexp doesn't match if they
					 * don't. */
					if n > len(inspect_ver){
						break
					}
					if v, err := strconv.ParseInt(inspect_ver[n], 10, 64); err != nil {
						verr := Errorf(err.Error())
						return "", verr
					} else {
						if v < 0 {
							verr := Errorf("Invalid version number")
							return "", verr
						}
					}
				}
			} else {
				verr := Errorf("Invalid version number")
				return "", verr
			}

			return ver, nil
		case nil:
			return "0.0.0", nil
		default:
			err := Errorf("Invalid version number")
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

func ValidateCookbookDivision(dname string, div interface{}) ([]map[string]interface{}, Gerror) {
	switch div := div.(type) {
		case []interface{}:
			d := make([]map[string]interface{}, 0)
			err := Errorf("Invalid element in array value of '%s'.", dname)
			
			for _, v := range div {
				switch v := v.(type){
					case map[string]interface{}:
						if len(v) < 4 {
							return nil, err
						}
						/* validate existence of file
						 * in sandbox */
						chksum, cherr := ValidateAsString(v["checksum"])
						if cherr == nil {
							if _, ferr := filestore.Get(chksum); ferr != nil {
								var merr Gerror
								/* This is nuts. */
								if dname == "recipes" {
									merr = Errorf("Manifest has a checksum that hasn't been uploaded.")
								} else {
									merr = Errorf("Manifest has checksum %s but it hasn't yet been uploaded", chksum)
								}
								return nil, merr
							}
							item_url := fmt.Sprintf("/file_store/%s", chksum)
							v["url"] = CustomURL(item_url)
							d = append(d, v)
						}
					default:
						return nil, err
				}
			}

			return d, nil
		case nil:
			/* This the way? */
			// d := make([]map[string]interface{}, 0)
			return nil, nil
		default:
			err := Errorf("Field '%s' invalid", dname)
			return nil, err
	}
}

func ValidateNumVersions(nr string) Gerror {
	/* Just see if it fits the bill for what we want. */
	if nr != "all" && nr != "" {
		valid_nr := regexp.MustCompile(`^\d+`)
		m := valid_nr.MatchString(nr)
		if !m {
			err := Errorf("Invalid num_versions")
			return err
		}
		n, nerr := strconv.Atoi(nr)
		if nerr != nil {
			err := Errorf(nerr.Error())
			return err
		}
		if n < 0 {
			err := Errorf("Invalid num_versions")
			return err
		}
	} else if nr == "" {
		err := Errorf("Invalid num_versions")
		return err
	}
	return nil
}

func ValidateCookbookMetadata(mdata interface{}) (map[string]interface{}, Gerror){
	switch mdata := mdata.(type) {
		case map[string]interface{}:
			if len(mdata) == 0 {
				/* This error message would make more sense as
				 * "Metadata empty" if the metadata is, you
				 * know, totally empty, but the spec wants
				 * "Field 'metadata.version' missing." Since
				 * it's easier to just check the length before
				 * doing a for loop, check the length first
				 * before inspecting each map key. We have to
				 * give it the error message it wants first
				 * however. */
				
				err := Errorf("Field 'metadata.version' missing")

				return nil, err
			}
			/* If metadata does have a length, loop through and
			 * check the various elements. Some metadata fields are
			 * supposed to be strings, some are supposed to be 
			 * hashes. Versions is it's own special thing, of
			 * course, and needs checked seperately. Do that first.
			 */
			if mv, mvok := mdata["version"]; mvok {
				switch mv := mv.(type) {
					case string:
						if _, merr := ValidateAsVersion(mv); merr != nil {
						merr := Errorf("Field 'metadata.version' invalid")
						return nil, merr
						}
					case nil:
						;
					default:
						err := Errorf("Field 'metadata.version' invalid")
						return nil, err
				}
			} else {
				err := Errorf("Field 'metadata.version' missing")
				return nil, err
			}

			/* String checks. Check equality of name and version
			 * elsewhere. */
			strchk := []string{ "maintainer", "name", "description", "maintainer_email", "long_description", "license" }
			for _, v := range strchk {
				err := Errorf("Field 'metadata.%s' invalid", v)
				switch sv := mdata[v].(type) {
					case string:
						if v == "name" && !ValidateEnvName(sv) {
							return nil, err
						}
						_ = sv // no-op
					case nil:
						if v == "long_description" {
							mdata[v] = ""
						} 
					default:
						return nil, err
				}
			}
			/* hash checks */
			hashchk := []string{ "platforms", "dependencies", "recommendations", "suggestions", "conflicting", "replacing", "groupings" }
			for _, v := range hashchk {
				err := Errorf("Field 'metadata.%s' invalid", v)
				switch hv := mdata[v].(type) {
					case map[string]interface{}:
						for _, j := range hv {
							switch s := j.(type) {
								case string:
									if _, serr := ValidateAsConstraint(s); serr != nil {
										cerr := Errorf("Invalid value '%s' for metadata.%s", s, v)
										return nil, cerr
									}
								case map[string]interface{}:
									if v != "groupings" {
										err := Errorf("Invalid value '{[]}' for metadata.%s", v)
										return nil, err
									}
								default:
									fakeout := fmt.Sprintf("%v", s)
									if fakeout == "map[]" {
										fakeout = "{[]}"
									}
									err := Errorf("Invalid value '%s' for metadata.%s", fakeout, v)
									return nil, err
							}
						}
					case nil:
						if v == "dependencies" {
							mdata[v] = make(map[string]interface{})
						}
					default:
						return nil, err
				}
			}

			return mdata, nil
		default:
			err := Errorf("bad metadata: chng msg")
			return nil, err
	}
}

func ValidateAsConstraint(t interface{}) (bool, Gerror) {
	err := Errorf("Invalid constraint")
	switch t := t.(type) {
		case string:
			cr := regexp.MustCompile(`^([<>=~]{1,2}) (.*)`)
			c_item := cr.FindStringSubmatch(t)
			if c_item != nil {
				ver := c_item[2]
				if _, verr := ValidateAsVersion(ver); verr != nil {
					return false, verr
				}
				return true, nil
			} else {
				return false, err
			}
		default:
			return false, err
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
		valid_ver := regexp.MustCompile(`^\d+\.\d+(\.\d+)?$`)
		if !valid_ver.MatchString(version){
			return false
		}
	}
	/* If we get this far, just do a final check on the name */
	final_chk := regexp.MustCompile(`^\w+(::\w+)?$`)
	n := final_chk.MatchString(name)

	return n
}
