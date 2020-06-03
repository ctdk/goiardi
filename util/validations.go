/*
 * Copyright (c) 2013-2017, Jeremy Bingham (<jeremy@goiardi.gl>)
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

package util

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/filestore"
	"github.com/tideland/golib/logger"
)

/* Validations for different types and input. */

func ValidateName(name string) bool {
	m, _ := regexp.MatchString("[^A-Za-z0-9_.-]", name)
	return !m
}

func ValidateUserName(name string) bool {
	m, _ := regexp.MatchString("[^a-z0-9_.-]", name)
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

func ValidateAsBool(b interface{}) (bool, Gerror) {
	switch b := b.(type) {
	case bool:
		return b, nil
	default:
		err := Errorf("Invalid bool")
		return false, err
	}
}

func ValidateAsFieldString(str interface{}) (string, Gerror) {
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

func ValidateAsVersion(ver interface{}) (string, Gerror) {
	switch ver := ver.(type) {
	case string:
		validVer := regexp.MustCompile(`^(\d+)\.(\d+)(\.?)(\d+)?$`)
		inspectVer := validVer.FindStringSubmatch(ver)

		if inspectVer != nil && ver != "0.0" {
			nums := []int{1, 2, 4}
			for _, n := range nums {
				/* #4 might not exist, but 1 and 2 must.
				 * The regexp doesn't match if they
				 * don't. */
				if n > len(inspectVer) || inspectVer[n] == "" && n == 4 {
					break
				}
				v, err := strconv.ParseInt(inspectVer[n], 10, 64)
				if err != nil {
					verr := Errorf(err.Error())
					return "", verr
				}
				if v < 0 {
					verr := Errorf("Invalid version number")
					return "", verr
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

func ValidateAttributes(key string, attrs interface{}) (map[string]interface{}, Gerror) {
	switch attrs := attrs.(type) {
	case map[string]interface{}:
		return attrs, nil
	case nil:
		/* Separate to do more validations above */
		nilAttrs := make(map[string]interface{})
		return nilAttrs, nil
	default:
		err := Errorf("Field '%s' is not a hash", key)
		return nil, err
	}
}

func ValidateCookbookDivision(dname string, div interface{}) ([]map[string]interface{}, Gerror) {
	switch div := div.(type) {
	case []interface{}:
		var d []map[string]interface{}
		err := Errorf("Invalid element in array value of '%s'.", dname)

		for _, v := range div {
			switch v := v.(type) {
			case map[string]interface{}:
				if len(v) < 4 {
					return nil, err
				}
				/* validate existence of file
				 * in sandbox */
				chksum, cherr := ValidateAsString(v["checksum"])
				if cherr == nil {
					var itemURL string
					var uploaded bool
					var ferr error

					if config.Config.UseS3Upload {
						uploaded, ferr = CheckForObject("default", chksum)
						if ferr != nil {
							uploaded = false
							logger.Errorf(ferr.Error())
						}
					} else {
						if _, ferr = filestore.Get(chksum); ferr == nil {
							uploaded = true
						}
					}
					//if file has not been uploaded return an error
					if !uploaded {
						var merr Gerror
						/* This is nuts. */
						if dname == "recipes" {
							merr = Errorf("Manifest has a checksum that hasn't been uploaded.")
						} else {
							merr = Errorf("Manifest has checksum %s but it hasn't yet been uploaded", chksum)
						}
						return nil, merr
					}

					//
					if config.Config.UseS3Upload && !config.Config.UseS3Proxy {
						itemURL, _ = S3GetURL("default", chksum)
					} else {
						itemURL = CustomURL(fmt.Sprintf("/file_store/%s", chksum))
					}

					v["url"] = itemURL
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
		validNr := regexp.MustCompile(`^\d+`)
		m := validNr.MatchString(nr)
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

func ValidateCookbookMetadata(mdata interface{}) (map[string]interface{}, Gerror) {
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
		strchk := []string{"maintainer", "name", "description", "maintainer_email", "long_description", "license"}
		for _, v := range strchk {
			err := Errorf("Field 'metadata.%s' invalid", v)
			switch sv := mdata[v].(type) {
			case string:
				if v == "name" && !ValidateName(sv) {
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
		hashchk := []string{"platforms", "dependencies", "recommendations", "suggestions", "conflicting", "replacing", "groupings"}
		for _, v := range hashchk {
			err := Errorf("Field 'metadata.%s' invalid", v)
			switch hv := mdata[v].(type) {
			case map[string]interface{}:
				for _, j := range hv {
					switch s := j.(type) {
					case string:
						if _, serr := ValidateAsConstraint(s); serr != nil {
							if _, serr = ValidateAsVersion(s); serr != nil {
								cerr := Errorf("Invalid value '%s' for metadata.%s", s, v)
								return nil, cerr
							}
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
		cItem := cr.FindStringSubmatch(t)
		if cItem != nil {
			ver := cItem[2]
			if _, verr := ValidateAsVersion(ver); verr != nil {
				return false, verr
			}
			return true, nil
		}
		return false, err
	default:
		return false, err
	}
}

func ValidateRunList(rl interface{}) ([]string, Gerror) {
	switch rl := rl.(type) {
	case []string:
		for i, r := range rl {
			j, err := validateRLItem(r)
			if err != nil {
				return nil, err
			}
			if j == "" {
				err := Errorf("Field 'run_list' is not a valid run list")
				return nil, err
			}
			rl[i] = j
		}

		/* Remove dupes */
		result := []string{}
		found := map[string]bool{}
		for _, u := range rl {
			if _, ok := found[u]; !ok {
				result = append(result, u)
				found[u] = true
			}
		}

		return result, nil
	case nil:
		/* separate to do more validations above */
		var nilRl []string
		return nilRl, nil
	default:
		err := Errorf("Not a proper runlist []string")
		return nil, err
	}
}

func validateRLItem(item string) (string, Gerror) {
	/* There's a few places this might be used. */
	err := Errorf("Field 'run_list' is not a valid run list")

	if item == "" {
		return "", err
	}

	/* first checks */
	validRl := regexp.MustCompile("[^A-Za-z0-9_\\[\\]@\\.:-]")
	m := validRl.MatchString(item)

	if m {
		return "", err
	}

	inspectRegexp := regexp.MustCompile(`^(\w+)\[(.*?)\]$`)
	inspectItem := inspectRegexp.FindStringSubmatch(item)

	if inspectItem != nil {
		rlType := inspectItem[1]
		rlItem := inspectItem[2]
		if rlType == "role" {
			if !validateRoleName(rlItem) {
				return "", err
			}
		} else if rlType == "recipe" {
			if !validateRecipeName(rlItem) {
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
	validRole := regexp.MustCompile("[^A-Za-z0-9_-]")
	m := validRole.MatchString(name)
	return !m
}

func validateRecipeName(name string) bool {
	firstValid := regexp.MustCompile("[^A-Za-z0-9_@\\.:-]")
	m := firstValid.MatchString(name)
	if m {
		return false
	}

	/* If we have a version */
	if strings.Index(name, "@") != -1 {
		h := strings.Split(name, "@")
		name = h[0]
		version := h[1]
		validVer := regexp.MustCompile(`^\d+\.\d+(\.\d+)?$`)
		if !validVer.MatchString(version) {
			return false
		}
	}
	return true
}

// CheckAdminPlusValidator checks that client/user json is not trying to set
// admin and validator at the same time. This has to be checked separately to
// make chef-pedent happy.
func CheckAdminPlusValidator(jsonActor map[string]interface{}) Gerror {
	var ab, vb bool
	if adminVal, ok := jsonActor["admin"]; ok {
		ab, _ = ValidateAsBool(adminVal)
	}
	if validatorVal, ok := jsonActor["validator"]; ok {
		vb, _ = ValidateAsBool(validatorVal)
	}
	if ab && vb {
		err := Errorf("Client can be either an admin or a validator, but not both.")
		err.SetStatus(http.StatusBadRequest)
		return err
	}
	return nil
}
