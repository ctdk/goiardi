/* Local file storage stuff, for when we just want to upload files locally and
 * not send them to s3 or somesuch. A building block of sandbox and cookbook
 * functionality. */

/*
 * Copyright (c) 2013-2014, Jeremy Bingham (<jbingham@gmail.com>)
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
	"io"
	"fmt"
	"github.com/ctdk/goiardi/data_store"
	"crypto/md5"
)

/* Local filestorage struct. Add fields as needed. */

type FileStore struct {
	Chksum string
	Data *[]byte
}

/* New, for this, includes giving it the file data */

func New(chksum string, data io.ReadCloser, data_length int64) (*FileStore, error){
	if _, err := Get(chksum); err == nil {
		err := fmt.Errorf("File with checksum %s already exists.", chksum)
		return nil, err
	}
	/* Read the data in */
	file_data := make([]byte, data_length)
	if n, err := io.ReadFull(data, file_data); err != nil {
		/* Something went wrong reading the data! */
		read_err := fmt.Errorf("Only read %d bytes (out of %d, supposedly) from io.ReadCloser: %s", n, data_length, err.Error())
		return nil, read_err
	}
	/* Verify checksum. May move to a different function later. */
	ver_chk := md5.New()
	/* try writestring first */
	ver_chk.Write(file_data)
	ver_chksum := fmt.Sprintf("%x", ver_chk.Sum(nil))
	if ver_chksum != chksum {
		chk_err := fmt.Errorf("Checksum %s did not match original %s!", ver_chksum, chksum)
		return nil, chk_err
	}
	filestore := &FileStore {
		Chksum: chksum,
		Data: &file_data,
	}
	return filestore, nil
}

func Get(chksum string) (*FileStore, error){
	ds := data_store.New()
	filestore, found := ds.Get("filestore", chksum)
	if !found {
		err := fmt.Errorf("File with checksum %s not found", chksum)
		return nil, err
	}
	return filestore.(*FileStore), nil
}

func (f *FileStore) Save() error {
	ds := data_store.New()
	ds.Set("filestore", f.Chksum, f)
	return nil
}

func (f *FileStore) Delete() error {
	ds := data_store.New()
	ds.Delete("filestore", f.Chksum)
	return nil
}

func GetList() []string {
	ds := data_store.New()
	file_list := ds.GetList("filestore")
	return file_list
}
