/*
 * Copyright (c) 2013-2016, Jeremy Bingham (<jeremy@goiardi.gl>)
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
	"sort"
	"testing"
)

func TestStringTrimming(t *testing.T) {
	s := "12345"
	u := "12345678901234567890"
	l := 8
	s = TrimStringMax(s, 8)
	u = TrimStringMax(u, 8)
	if len(s) != 5 {
		t.Errorf("post-trim len for s should have been 5, somehow got %d", len(s))
	}
	if len(u) != l {
		t.Errorf("post-trim len for u should have been %d, got %d (%s)", l, len(u), u)
	}
}

func TestDupRemoval(t *testing.T) {
	strs := []string{ "This", "", "has", "", "some", "", "some", "dupes" }
	sort.Strings(strs)
	strs = RemoveDupStrings(strs)
	chkmap := make(map[string]uint8)
	for _, v := range strs {
		chkmap[v]++
	}
	for k, v := range chkmap {
		if v > 1 {
			t.Errorf("string '%s' had %d elements, should have had 1", k, v)
		}
	}
}
