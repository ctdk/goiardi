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
)

// Anything that implements these functions is a goiardi/chef object, like a
// cookbook, role, etc., and will be able to use these common functions.
type GoiardiObj interface {
	GetName() string
	URLType() string
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
