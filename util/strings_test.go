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

func TestStringTrimmingNoTrim(t *testing.T) {
	faz := "hello there"
	boo := "oogety boogety"
	l := 0
	fazlen := len(faz)
	boolen := len(boo)
	faz = TrimStringMax(faz, l)
	boo = TrimStringMax(boo, l)
	if len(faz) != fazlen {
		t.Errorf("post-trim len for 'faz' should have been the same at %d, but instead we got %d", fazlen, len(faz))
	}
	if len(boo) != boolen {
		t.Errorf("post-trim len for 'boo' should have been the same at %d, but instead we got %d", boolen, len(boo))
	}
}

func TestStringTrimmingUTF8(t *testing.T) {
	u := "123456üøöß≈ç"
	q := "öåπi"
	l := 8
	u = TrimStringMax(u, l)
	if len(u) == l {
		t.Errorf("post-trim len for u with unicode characters should not be %d", l)
	}
	if len([]rune(u)) != l {
		t.Errorf("trimmed string length in runes should have been %d, but was %d", l, len([]rune(u)))
	}

	oq := len(q)
	orq := len([]rune(q))
	q = TrimStringMax(q, l)
	if len(q) != oq {
		t.Errorf("post-trim len for q with unicode characters shorter than the specified length '%d' should have stayed the same, but changed. Expected %d, got %d.", l, oq, len(q))
	}
	if len([]rune(q)) != orq {
		t.Errorf("post-trim rune array len for q with unicode characters shorter than the specified length '%d' should have stayed the same, but changed. Expected %d, got %d.", l, orq, len([]rune(q)))
	}

}

func TestDupRemoval(t *testing.T) {
	strs := []string{"This", "", "has", "", "some", "", "some", "dupes"}
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
