/*
 * Copyright (c) 2013-2026, Jeremy Bingham (<jeremy@goiardi.gl>)
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

package filestore

import (
	"bytes"
	"crypto/md5"
	"encoding/gob"
	"fmt"
	"io"
	"testing"

	"github.com/ctdk/goiardi/datastore"
)

func init() {
	gob.Register(new(FileStore))
}

type fakeOrg struct {
	name string
	id   int64
}

func (f *fakeOrg) GetName() string { return f.name }
func (f *fakeOrg) GetId() int64    { return f.id }

var filestoreOrgCounter int

func setupFilestoreTestOrg(t *testing.T) FstoreOrg {
	filestoreOrgCounter++
	return &fakeOrg{name: fmt.Sprintf("testfilestore-%d", filestoreOrgCounter), id: int64(filestoreOrgCounter)}
}

func checksum(data []byte) string {
	h := md5.New()
	h.Write(data)
	return fmt.Sprintf("%x", h.Sum(nil))
}

func makeReadCloser(data []byte) io.ReadCloser {
	return io.NopCloser(bytes.NewReader(data))
}

func TestFilestoreNew(t *testing.T) {
	org := setupFilestoreTestOrg(t)
	data := []byte("hello filestore")
	chksum := checksum(data)
	fs, err := New(org, chksum, makeReadCloser(data), int64(len(data)))
	if err != nil {
		t.Fatalf("New() failed: %s", err.Error())
	}
	if fs.Chksum != chksum {
		t.Errorf("expected checksum %s, got %s", chksum, fs.Chksum)
	}
	if !bytes.Equal(*fs.Data, data) {
		t.Errorf("data did not match")
	}
}

func TestFilestoreNewBadChecksum(t *testing.T) {
	org := setupFilestoreTestOrg(t)
	data := []byte("hello filestore")
	_, err := New(org, "deadbeefdeadbeefdeadbeefdeadbeef", makeReadCloser(data), int64(len(data)))
	if err == nil {
		t.Fatal("expected error for mismatched checksum")
	}
}

func TestFilestoreSaveAndGet(t *testing.T) {
	org := setupFilestoreTestOrg(t)
	data := []byte("save me")
	chksum := checksum(data)
	fs, _ := New(org, chksum, makeReadCloser(data), int64(len(data)))
	if err := fs.Save(); err != nil {
		t.Fatalf("Save() failed: %s", err.Error())
	}
	fs2, err := Get(org, chksum)
	if err != nil {
		t.Fatalf("Get() failed: %s", err.Error())
	}
	if fs2.Chksum != chksum {
		t.Errorf("expected checksum %s, got %s", chksum, fs2.Chksum)
	}
	if !bytes.Equal(*fs2.Data, data) {
		t.Errorf("retrieved data did not match")
	}
}

func TestFilestoreGetNotFound(t *testing.T) {
	org := setupFilestoreTestOrg(t)
	_, err := Get(org, "deadbeefdeadbeefdeadbeefdeadbeef")
	if err == nil {
		t.Fatal("expected error for missing filestore item")
	}
}

func TestFilestoreDuplicateSameData(t *testing.T) {
	org := setupFilestoreTestOrg(t)
	data := []byte("dup data")
	chksum := checksum(data)
	fs, _ := New(org, chksum, makeReadCloser(data), int64(len(data)))
	fs.Save()
	fs2, err := New(org, chksum, makeReadCloser(data), int64(len(data)))
	if err != nil {
		t.Fatalf("New() with duplicate same data failed: %s", err.Error())
	}
	if fs2 == nil {
		t.Fatal("expected FileStore for duplicate same data, got nil")
	}
}

func TestFilestoreDuplicateDifferentData(t *testing.T) {
	org := setupFilestoreTestOrg(t)
	data1 := []byte("data one")
	data2 := []byte("data two")
	chksum := checksum(data1)
	fs, _ := New(org, chksum, makeReadCloser(data1), int64(len(data1)))
	fs.Save()
	_, err := New(org, chksum, makeReadCloser(data2), int64(len(data2)))
	if err == nil {
		t.Fatal("expected error for duplicate with different data")
	}
}

func TestFilestoreDelete(t *testing.T) {
	org := setupFilestoreTestOrg(t)
	data := []byte("delete me")
	chksum := checksum(data)
	fs, _ := New(org, chksum, makeReadCloser(data), int64(len(data)))
	fs.Save()
	if err := fs.Delete(); err != nil {
		t.Fatalf("Delete() failed: %s", err.Error())
	}
	_, err := Get(org, chksum)
	if err == nil {
		t.Fatal("expected error getting deleted filestore item")
	}
}

func TestFilestoreGetList(t *testing.T) {
	org := setupFilestoreTestOrg(t)
	for _, d := range []string{"a", "b", "c"} {
		data := []byte(d)
		chksum := checksum(data)
		fs, _ := New(org, chksum, makeReadCloser(data), int64(len(data)))
		fs.Save()
	}
	list := GetList(org)
	if len(list) != 3 {
		t.Errorf("expected 3 filestore items, got %d", len(list))
	}
}

func TestFilestoreAllFilestores(t *testing.T) {
	org := setupFilestoreTestOrg(t)
	for _, d := range []string{"all-a", "all-b"} {
		data := []byte(d)
		chksum := checksum(data)
		fs, _ := New(org, chksum, makeReadCloser(data), int64(len(data)))
		fs.Save()
	}
	all := AllFilestores(org)
	if len(all) != 2 {
		t.Errorf("expected 2 filestore items, got %d", len(all))
	}
}

func TestFilestoreDeleteHashes(t *testing.T) {
	org := setupFilestoreTestOrg(t)
	var hashes []string
	for _, d := range []string{"del-a", "del-b"} {
		data := []byte(d)
		chksum := checksum(data)
		hashes = append(hashes, chksum)
		fs, _ := New(org, chksum, makeReadCloser(data), int64(len(data)))
		fs.Save()
	}
	DeleteHashes(org, hashes)
	for _, chksum := range hashes {
		if _, err := Get(org, chksum); err == nil {
			t.Errorf("expected %s to be deleted", chksum)
		}
	}
}

func TestFilestoreActionAtADistance(t *testing.T) {
	org := setupFilestoreTestOrg(t)
	data := []byte("action")
	chksum := checksum(data)
	fs, _ := New(org, chksum, makeReadCloser(data), int64(len(data)))
	fs.Save()
	fs2, _ := Get(org, chksum)
	(*fs2.Data)[0] = 'X'
	fs3, _ := Get(org, chksum)
	if (*fs3.Data)[0] == 'X' {
		t.Error("modifying retrieved filestore data affected the stored copy")
	}
	_ = datastore.New()
}
