/*
 * Copyright (c) 2013-2018, Jeremy Bingham (<jeremy@goiardi.gl>)
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

package sandbox

import (
	"crypto/md5"
	"encoding/gob"
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/fakeacl"
	"github.com/ctdk/goiardi/indexer"
	"github.com/ctdk/goiardi/organization"
	"fmt"
	"math/rand"
	"testing"
	"time"
)

const (
	randStrLen   = 20
	numChecksums = 7
)

// borrowing this from Stack Overflow (such as it ever is), located at
// https://stackoverflow.com/questions/22892120/how-to-generate-a-random-string-of-a-fixed-length-in-golang

var src = rand.NewSource(time.Now().UnixNano())

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

func init() {
	indexer.Initialize(config.Config)
}

func randStringBytesMaskImprSrc(n int) string {
	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return string(b)
}

func randomHashes(num int) map[string]interface{} {
	h := make(map[string]interface{}, num)
	for i := 0; i < num; i++ {
		s := randStringBytesMaskImprSrc(randStrLen)
		chksum := md5.Sum([]byte(s))
		ascii := fmt.Sprintf("%x", chksum)
		h[ascii] = nil
	}
	return h
}

func TestSandboxPurgeWith3(t *testing.T) {
	ss := new(Sandbox)
	gob.Register(ss)
	gob.Register(new(organization.Organization))
	org, _ := organization.New("sboxpurge", "sboxpurge")
	fakeacl.LoadFakeACL(org)
	org.Save()

	tm := time.Now()
	cs1 := randomHashes(numChecksums)
	cs2 := randomHashes(numChecksums)
	cs3 := randomHashes(numChecksums)

	sb1, err := New(org, cs1)
	if err != nil {
		t.Error(err)
	}
	sb2, err := New(org, cs2)
	if err != nil {
		t.Error(err)
	}
	sb3, err := New(org, cs3)
	if err != nil {
		t.Error(err)
	}

	// Make one of the sandboxes pretend to be old
	sb1.CreationTime = tm.Add(-7 * 24 * time.Hour)
	sb1.Save()
	sb2.Save()
	sb3.Save()

	olderThan := 5 * 24 * time.Hour
	d, err := Purge(olderThan)
	if err != nil {
		t.Error(err)
	}
	if d != 1 {
		t.Errorf("One sandbox should have been deleted, but %d were purged.", d)
	}

	all := AllSandboxes(org)
	if len(all) != 2 {
		t.Errorf("After purging there should have been 2 sandboxes, but there are %d.", len(all))
	}
	sb2.Delete()
	sb3.Delete()
}

func TestSandboxPurgeWith30(t *testing.T) {
	org, _ := organization.New("sboxpurge30", "sboxpurge30")
	fakeacl.LoadFakeACL(org)
	org.Save()

	tm := time.Now()

	slen := 30
	for si := 0; si < slen; si++ {
		h := randomHashes(numChecksums)
		sb, _ := New(org, h)
		if (si % 5) == 0 {
			sb.CreationTime = tm.Add(-7 * 24 * time.Hour)
		}
		sb.Save()
	}
	olderThan := 5 * 24 * time.Hour
	d, _ := Purge(olderThan)

	if d != 6 {
		t.Errorf("should have purged 6 sandboxes, actually purged %d", d)
	}
	all := AllSandboxes(org)
	if len(all) != 24 {
		t.Errorf("After purging there should have been 24 sandboxes, but there were %d instead", len(all))
	}

}
