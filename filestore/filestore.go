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

// Package filestore provides local file uploads and downloads for cookbook
// uploading and downloading. All access to the files is through the checksum,
// rather than the file name.
//
// If config.Config.LocalFstoreDir is != "", the content of the files will be
// stored in that directory.
package filestore

import (
	"io"
	"fmt"
	"github.com/ctdk/goiardi/data_store"
	"crypto/md5"
	"github.com/ctdk/goiardi/config"
	"database/sql"
	"os"
	"path"
	"git.tideland.biz/goas/logger"
)

/* Local filestorage struct. Add fields as needed. */

// An individual file in the filestore. Note that there is no actual name for
// the file used, but it is identified by the file's checksum. The file's data
// is stored as a pointer to an array of bytes.
type FileStore struct {
	Chksum string
	Data *[]byte
}

/* New, for this, includes giving it the file data */

// Create a new filestore item with the given checksum, io.ReadCloser holding
// the file's data, and the length of the file. If the file data's checksum does
// not match the provided checksum an error will be trhown.
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
	var filestore *FileStore
	var found bool
	if config.Config.UseMySQL {
		var err error
		filestore, err = getMySQL(chksum)
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
		ds := data_store.New()
		var f interface{}
		f, found = ds.Get("filestore", chksum)
		if f != nil {
			filestore = f.(*FileStore)
		}
	}
	if !found {
		err := fmt.Errorf("File with checksum %s not found", chksum)
		return nil, err
	}
	if config.Config.LocalFstoreDir != "" {
		/* File data is stored on disk */
		chkPath := path.Join(config.Config.LocalFstoreDir, chksum)
		
		fp, err := os.Open(chkPath)
		if err != nil {
			return nil, err
		}
		defer fp.Close()
		stat, sterr := fp.Stat()
		if sterr != nil {
			return nil, sterr
		}
		fdata := make([]byte, stat.Size())
		n, fperr := fp.Read(fdata)
		if fperr != nil {
			return nil, fperr
		} else if int64(n) != stat.Size() {
			err = fmt.Errorf("only %d bytes were read from the expected %d", n, stat.Size())
			return nil, err
		}
		filestore.Data = &fdata
	}
	return filestore, nil
}

func (f *FileStore) Save() error {
	if config.Config.UseMySQL {
		err := f.saveMySQL()
		if err != nil {
			return err
		}
	} else {
		ds := data_store.New()
		ds.Set("filestore", f.Chksum, f)
	}
	if config.Config.LocalFstoreDir != "" {
		fp, err := os.Create(path.Join(config.Config.LocalFstoreDir, f.Chksum))
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

func (f *FileStore) Delete() error {
	if config.Config.UseMySQL {
		err := f.deleteMySQL()
		if err != nil {
			return err
		}
	} else {
		ds := data_store.New()
		ds.Delete("filestore", f.Chksum)
	}

	if config.Config.LocalFstoreDir != "" {
		err := os.Remove(path.Join(config.Config.LocalFstoreDir, f.Chksum))
		if err != nil {
			return err
		}
	}
	return nil
}

// Get a list of files that have been uploaded.
func GetList() []string {
	var file_list []string
	if config.Config.UseMySQL {
		file_list = getListMySQL()
	} else {
		ds := data_store.New()
		file_list = ds.GetList("filestore")
	}
	return file_list
}

// Delete all the checksum hashes given from the filestore.
func DeleteHashes(file_hashes []string) {
	if config.Config.UseMySQL {
		deleteHashesMySQL(file_hashes)
	} else {
		for _, ff := range file_hashes {
		del_file, err := Get(ff)
			if err != nil {
				logger.Debugf("Strange, we got an error trying to get %s to delete it.\n", ff)
				logger.Debugf(err.Error())
			} else {
				_ = del_file.Delete()
			}
			// May be able to remove this. Check that it actually deleted
			d, _ := Get(ff)
			if d != nil {
				logger.Debugf("Stranger and stranger, %s is still in the file store.\n", ff)
			}
		}
	}
	if config.Config.LocalFstoreDir != "" {
		for _, fh := range file_hashes {
			err := os.Remove(path.Join(config.Config.LocalFstoreDir, fh))
			if err != nil {
				logger.Errorf(err.Error())
			}
		}
	}
}

func AllFilestores() []*FileStore {
	filestores := make([]*FileStore, 0)
	if config.Config.UseMySQL {
		filestores = allFilestoresSQL()
	} else {
		file_list := GetList()
		for _, f := range file_list {
			fl, err := Get(f)
			if err != nil {
				logger.Debugf("File checksum %s was in the list of files, but wasn't found when fetched. Continuing.", f)
				continue
			}
			filestores = append(filestores, fl)
		}
	}
	return filestores
}
