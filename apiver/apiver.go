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

// Package apiver handles Chef Server API related tasks and checks. There are
// some meaningful differences between version 0 and version 1 than some
// endpoints need to handle correctly.
package apiver

import (
	"github.com/ctdk/goiardi/gerror"
	"github.com/ctdk/goiardi/logger"
	"net/http"
	"strconv"
	"strings"
)

// move the goiardi & chef server versions in here later too?

const ChefApiVersion = "0"

// Minimum supported API version
const MinAPIVersion = "0"

// Maximum supported API version
const MaxAPIVersion = "2"

// Default API version if no header or API version provided.
const DefaultAPIVersion = "1"

const APIv0 = 0
const APIv1 = 1
const APIv2 = 2 // there definitely is a v2 API, but it's still mysterious to me

// All supported API versions
var SupportedAPIVersions = []string{"0", "1", "2"}

// MatchSupportedAPIVersion compares an API version requested from the headers
// to the versions supported by this server.
func MatchSupportedAPIVersion(ver string) bool {
	// if it's empty, it should match the default API version
	if ver == "" {
		return true
	}
	for _, v := range SupportedAPIVersions {
		if ver == v {
			return true
		}
	}
	return false
}

// GetReqAPIVersion gets the request's desired API version from the headers and
// returns an int for easy checking in a handler.
func GetReqAPIVersion(r *http.Request) (int, gerror.Error) {
	v := r.Header.Get("X-Ops-Server-API-Version")
	logger.Debugf("Requested Server API version: '%s'", v)
	if v == "" {
		v = DefaultAPIVersion
		logger.Debugf("Requested empty Server API version or the header was missing. Setting API to v%s.", v)
	}

	apiVer, err := strconv.Atoi(v)
	if err != nil {
		gerr := gerror.Errorf("Invalid X-Ops-Server-API-Version header '%s', error converting to string: %w", v, err)
		return 0, gerr
	}

	// Assuming for the moment that we ought to check the supported versions
	// and return some sort of error if it doesn't match. It's possible that
	// we shouldn't, but we'll see.
	if !MatchSupportedAPIVersion(v) {
		gerr := gerror.Errorf("Unsupported Server API Version '%s'. Supported versions are %s", v, strings.Join(SupportedAPIVersions, ", "))
		return 0, gerr
	}

	return apiVer, nil
}
