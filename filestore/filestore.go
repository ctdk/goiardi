/* Local file storage stuff, for when we just want to upload files locally and
 * not send them to s3 or somesuch. A building block of sandbox and cookbook
 * functionality. */

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

// Package filestore provides local file uploads and downloads for cookbook
// uploading and downloading. All access to the files is through the checksum,
// rather than the file name.
//
// If config.Config.LocalFstoreDir is != "", the content of the files will be
// stored in that directory.
package filestore

import (
	"bytes"
	"crypto/md5"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/datastore"
	"github.com/tideland/golib/logger"
)

/* Local filestorage struct. Add fields as needed. */

// FileStore is an individual file in the filestore. Note that there is no
// actual name for the file used, but it is identified by the file's checksum.
// The file's data is stored as a pointer to an array of bytes.
type FileStore struct {
	Chksum  string
	Data    *[]byte
	orgName string
}

/* New, for this, includes giving it the file data */

// New creates a new filestore item with the given checksum, io.ReadCloser
// holding the file's data, and the length of the file. If the file data's
// checksum does not match the provided checksum an error will be trhown.
func New(orgName string, chksum string, data io.ReadCloser, dataLength int64) (*FileStore, error) {
	f, err := Get(orgName, chksum)
	if err == nil {
		// if err is nil, wait until checking the uploaded content to
		// see if it's the same as what we have already
		err = fmt.Errorf("File with checksum %s already exists.", chksum)
	}
	/* Read the data in */
	fileData := make([]byte, dataLength)
	if n, err := io.ReadFull(data, fileData); err != nil {
		/* Something went wrong reading the data! */
		readErr := fmt.Errorf("Only read %d bytes (out of %d, supposedly) from io.ReadCloser: %s", n, dataLength, err.Error())
		return nil, readErr
	}
	if f != nil {
		if !bytes.Equal(fileData, *f.Data) {
			return nil, err
		}
	}
	/* Verify checksum. May move to a different function later. */
	verChk := md5.New()
	/* try writestring first */
	verChk.Write(fileData)
	verChksum := fmt.Sprintf("%x", verChk.Sum(nil))
	if verChksum != chksum {
		chkErr := fmt.Errorf("Checksum %s did not match original %s!", verChksum, chksum)
		return nil, chkErr
	}
	filestore := &FileStore{
		Chksum:  chksum,
		Data:    &fileData,
		orgName: orgName,
	}
	return filestore, nil
}

// Get the file with this checksum.
func Get(orgName string, chksum string) (*FileStore, error) {
	var filestore *FileStore
	var found bool
	if config.UsingDB() {
		var err error
		filestore, err = getSQL(chksum)
		if err != nil {
			if err == sql.ErrNoRows {
				found = false
			} else {
				return nil, err
			}
		} else {
			found = true
		}
	} else {
		ds := datastore.New()
		var f interface{}
		f, found = ds.Get(dataKey(orgName), chksum)
		if f != nil {
			filestore = f.(*FileStore)
			filestore.orgName = orgName
		}
	}
	if !found {
		err := fmt.Errorf("File with checksum %s not found", chksum)
		return nil, err
	}
	if config.Config.LocalFstoreDir != "" {
		if err := filestore.loadData(); err != nil {
			return nil, err
		}
	}

	if filestore.Data == nil {
		d := make([]byte, 0)
		filestore.Data = &d
	}

	return filestore, nil
}

func (f *FileStore) loadData() error {
	/* If this is called, file data is stored on disk */
	chkPath := path.Join(config.Config.LocalFstoreDir, f.orgName, f.Chksum)

	fp, err := os.Open(chkPath)
	if err != nil {
		return err
	}
	defer fp.Close()
	stat, sterr := fp.Stat()
	if sterr != nil {
		return sterr
	}
	fdata := make([]byte, stat.Size())
	n, fperr := fp.Read(fdata)
	if fperr != nil {
		return fperr
	} else if int64(n) != stat.Size() {
		err = fmt.Errorf("only %d bytes were read from the expected %d", n, stat.Size())
		return err
	}
	f.Data = &fdata
	return nil
}

// Save a file store item.
func (f *FileStore) Save() error {
	if config.Config.UseMySQL {
		err := f.saveMySQL()
		if err != nil {
			return err
		}
	} else if config.Config.UsePostgreSQL {
		err := f.savePostgreSQL()
		if err != nil {
			return nil
		}
	} else {
		ds := datastore.New()
		ds.Set(dataKey(f.orgName), f.Chksum, f)
	}
	if config.Config.LocalFstoreDir != "" {
		fp, err := os.Create(path.Join(config.Config.LocalFstoreDir, f.orgName, f.Chksum))
		if err != nil {
			return err
		}
		defer fp.Close()
		_, err = fp.Write(*f.Data)
		if err != nil {
			return err
		}
		return fp.Close()
	}
	return nil
}

// Delete a file store item.
func (f *FileStore) Delete() error {
	if config.UsingDB() {
		err := f.deleteSQL()
		if err != nil {
			return err
		}
	} else {
		ds := datastore.New()
		ds.Delete(dataKey(f.orgName), f.Chksum)
	}

	if config.Config.LocalFstoreDir != "" {
		err := os.Remove(path.Join(config.Config.LocalFstoreDir, f.orgName, f.Chksum))
		if err != nil {
			return err
		}
	}
	return nil
}

// GetList gets a list of files that have been uploaded.
func GetList(orgName string) []string {
	var fileList []string
	if config.UsingDB() {
		fileList = getListSQL()
	} else {
		ds := datastore.New()
		fileList = ds.GetList(dataKey(orgName))
	}
	return fileList
}

// DeleteHashes deletes all the checksum hashes given from the filestore.
func DeleteHashes(orgName string, fileHashes []string) {
	if config.Config.UseMySQL {
		deleteHashesMySQL(fileHashes)
	} else if config.Config.UsePostgreSQL {
		deleteHashesPostgreSQL(fileHashes)
	} else {
		for _, ff := range fileHashes {
			delFile, err := Get(orgName, ff)
			if err != nil {
				logger.Debugf("Strange, we got an error trying to get %s to delete it.\n", ff)
				logger.Debugf(err.Error())
			} else {
				_ = delFile.Delete()
			}
			// May be able to remove this. Check that it actually deleted
			d, _ := Get(orgName, ff)
			if d != nil {
				logger.Debugf("Stranger and stranger, %s is still in the file store.\n", ff)
			}
		}
	}
	if config.Config.LocalFstoreDir != "" {
		for _, fh := range fileHashes {
			err := os.Remove(path.Join(config.Config.LocalFstoreDir, orgName, fh))
			if err != nil {
				logger.Errorf(err.Error())
			}
		}
	}
}

// AllFilestores returns all file checksums and their contents, for exporting.
func AllFilestores(orgName string) []*FileStore {
	var filestores []*FileStore
	if config.UsingDB() {
		filestores = allFilestoresSQL()
	} else {
		fileList := GetList(orgName)
		filestores = make([]*FileStore, 0, len(fileList))
		for _, f := range fileList {
			fl, err := Get(orgName, f)
			if err != nil {
				logger.Debugf("File checksum %s was in the list of files, but wasn't found when fetched. Continuing.", f)
				continue
			}
			filestores = append(filestores, fl)
		}
	}
	return filestores
}

func dataKey(orgName string) string {
	return strings.Join([]string{"filestore-", orgName}, "")
}
