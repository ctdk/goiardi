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

package apiver

import (
	"testing"
)

func TestMatchSupportedAPIVersionV0(t *testing.T) {
	v := "0"
	if !MatchSupportedAPIVersion(v) {
		t.Errorf("version v%s should have matched a supported version, but didn't", v)
	}
}

func TestMatchSupportedAPIVersionV1(t *testing.T) {
	v := "1"
	if !MatchSupportedAPIVersion(v) {
		t.Errorf("version v%s should have matched a supported version, but didn't", v)
	}
}

func TestMatchSupportedAPIVersionV2(t *testing.T) {
	v := "2"
	if !MatchSupportedAPIVersion(v) {
		t.Errorf("version v%s should have matched a supported version, but didn't", v)
	}
}

func TestMatchSupportedAPIVersionVempty(t *testing.T) {
	v := ""
	if !MatchSupportedAPIVersion(v) {
		t.Error("An empty version should have matched a supported version, but didn't")
	}
}

func TestMatchSupportedAPIVersionVinvalid(t *testing.T) {
	v := "312"
	if MatchSupportedAPIVersion(v) {
		t.Errorf("version v%s should not have matched a supported version, but did", v)
	}
}

// TODO: craft fake http request object and test GetReqAPIVersion
