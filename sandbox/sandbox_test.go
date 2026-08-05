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

package sandbox

import (
	"bytes"
	"crypto/md5"
	"encoding/gob"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/fakeacl"
	"github.com/ctdk/goiardi/filestore"
	"github.com/ctdk/goiardi/indexer"
	"github.com/ctdk/goiardi/organization"
	"math/rand"
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
	indexer.Initialize(config.Config, indexer.DefaultDummyOrg)
	gob.Register(new(filestore.FileStore))
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
	tm := time.Now()

	org, _ := organization.New("sboxpurge30", "sboxpurge30")
	fakeacl.LoadFakeACL(org)
	org.Save()

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

func setupSandboxTest(t *testing.T) (*organization.Organization, func()) {
	t.Helper()
	tmpDir, err := ioutil.TempDir("", "sandbox-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %s", err.Error())
	}
	config.Config.PolicyRoot = tmpDir
	o, gerr := organization.New("sandbox-test-"+t.Name(), "Sandbox Test")
	if gerr != nil {
		t.Fatalf("failed to create org: %s", gerr.Error())
	}
	fakeacl.LoadFakeACL(o)
	if err := o.Save(); err != nil {
		t.Fatalf("failed to save org: %s", err.Error())
	}
	indexer.Initialize(config.Config, o)
	return o, func() { os.RemoveAll(tmpDir) }
}

func checksum(data []byte) string {
	h := md5.New()
	h.Write(data)
	return fmt.Sprintf("%x", h.Sum(nil))
}

func makeReadCloser(data []byte) io.ReadCloser {
	return io.NopCloser(bytes.NewReader(data))
}

func TestSandboxNew(t *testing.T) {
	org, cleanup := setupSandboxTest(t)
	defer cleanup()

	chk := checksum([]byte("test-data"))
	sb, err := New(org, map[string]interface{}{chk: nil})
	if err != nil {
		t.Fatalf("New failed: %s", err.Error())
	}
	if sb.ID == "" {
		t.Error("expected sandbox ID to be set")
	}
	if len(sb.Checksums) != 1 || sb.Checksums[0] != chk {
		t.Errorf("expected checksums [%s], got %v", chk, sb.Checksums)
	}
}

func TestSandboxNewInvalidChecksum(t *testing.T) {
	org, cleanup := setupSandboxTest(t)
	defer cleanup()

	if _, err := New(org, map[string]interface{}{"not-hex!!": nil}); err == nil {
		t.Error("expected error for invalid checksum")
	}
}

func TestSandboxSaveAndGet(t *testing.T) {
	org, cleanup := setupSandboxTest(t)
	defer cleanup()

	chk := checksum([]byte("test-data"))
	sb, _ := New(org, map[string]interface{}{chk: nil})
	if err := sb.Save(); err != nil {
		t.Fatalf("Save failed: %s", err.Error())
	}
	got, err := Get(org, sb.ID)
	if err != nil {
		t.Fatalf("Get failed: %s", err.Error())
	}
	if got.ID != sb.ID {
		t.Errorf("expected %s, got %s", sb.ID, got.ID)
	}
}

func TestSandboxDelete(t *testing.T) {
	org, cleanup := setupSandboxTest(t)
	defer cleanup()

	chk := checksum([]byte("test-data"))
	sb, _ := New(org, map[string]interface{}{chk: nil})
	sb.Save()
	if err := sb.Delete(); err != nil {
		t.Fatalf("Delete failed: %s", err.Error())
	}
	if _, err := Get(org, sb.ID); err == nil {
		t.Error("expected error after deleting sandbox")
	}
}

func TestSandboxGetListAndAllSandboxes(t *testing.T) {
	org, cleanup := setupSandboxTest(t)
	defer cleanup()

	chk1 := checksum([]byte("one"))
	chk2 := checksum([]byte("two"))
	sb1, _ := New(org, map[string]interface{}{chk1: nil})
	sb2, _ := New(org, map[string]interface{}{chk2: nil})
	sb1.Save()
	sb2.Save()

	list := GetList(org)
	if len(list) != 2 {
		t.Errorf("expected 2 sandboxes in list, got %d", len(list))
	}
	all := AllSandboxes(org)
	if len(all) != 2 {
		t.Errorf("expected 2 sandboxes from AllSandboxes, got %d", len(all))
	}
}

func TestSandboxUploadChkList(t *testing.T) {
	org, cleanup := setupSandboxTest(t)
	defer cleanup()

	data := []byte("uploaded")
	chk := checksum(data)
	other := checksum([]byte("missing"))
	fs, err := filestore.New(org, chk, makeReadCloser(data), int64(len(data)))
	if err != nil {
		t.Fatalf("failed to create filestore item: %s", err.Error())
	}
	if err := fs.Save(); err != nil {
		t.Fatalf("failed to save filestore item: %s", err.Error())
	}

	sb, _ := New(org, map[string]interface{}{chk: nil, other: nil})
	stats := sb.UploadChkList()
	if stats[chk]["needs_upload"] != false {
			t.Errorf("expected existing checksum not to need upload")
	}
	if stats[other]["needs_upload"] != true {
			t.Errorf("expected missing checksum to need upload")
	}
}

func TestSandboxIsComplete(t *testing.T) {
	org, cleanup := setupSandboxTest(t)
	defer cleanup()

	data := []byte("complete")
	chk := checksum(data)
	fs, err := filestore.New(org, chk, makeReadCloser(data), int64(len(data)))
	if err != nil {
		t.Fatalf("failed to create filestore item: %s", err.Error())
	}
	if err := fs.Save(); err != nil {
		t.Fatalf("failed to save filestore item: %s", err.Error())
	}

	sb, _ := New(org, map[string]interface{}{chk: nil})
	if err := sb.IsComplete(); err != nil {
		t.Fatalf("IsComplete failed for uploaded checksum: %s", err.Error())
	}

	sb2, _ := New(org, map[string]interface{}{checksum([]byte("missing")): nil})
	if err := sb2.IsComplete(); err == nil {
		t.Error("expected IsComplete to fail when checksum is missing")
	}
}
