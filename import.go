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

package main

import (
	"encoding/json"
	"encoding/base64"
	"bytes"
	"github.com/ctdk/goiardi/client"
	"github.com/ctdk/goiardi/cookbook"
	"github.com/ctdk/goiardi/data_bag"
	"github.com/ctdk/goiardi/environment"
	"github.com/ctdk/goiardi/filestore"
	// "github.com/ctdk/goiardi/log_info"
	// "github.com/ctdk/goiardi/node"
	// "github.com/ctdk/goiardi/report"
	// "github.com/ctdk/goiardi/role" 
	// "github.com/ctdk/goiardi/sandbox"
	"github.com/ctdk/goiardi/user" 
	"os"
	"io/ioutil"
	//"time"
	"fmt"
	"git.tideland.biz/goas/logger"
)

func importAll(fileName string) error {
	fp, err := os.Open(fileName)
	if err != nil {
		return err
	}
	exportedData := &ExportData{}
	dec := json.NewDecoder(fp)
	if err := dec.Decode(&exportedData); err != nil {
		return err
	}

	// What versions of the exported data are supported?
	// At the moment it's only 1.0.

	if exportedData.MajorVersion == 1 && exportedData.MinorVersion == 0 {
		logger.Infof("Importing data, version %d.%d created on %s", exportedData.MajorVersion, exportedData.MinorVersion, exportedData.CreatedTime)
		// so what do we have?
		// logger.Infof("dump\n%v", exportedData)
		// load clients
		for _, v := range exportedData.Data["client"] {
			if c, err := client.NewFromJson(v.(map[string]interface{})); err != nil {
				return err
			} else {
				gerr := c.Save()
				if gerr != nil {
					return gerr
				}
			}
		}

		// load users
		for _, v := range exportedData.Data["user"] {
			if u, err := user.NewFromJson(v.(map[string]interface{})); err != nil {
				return err
			} else {
				gerr := u.Save()
				if gerr != nil {
					return gerr
				}
			}
		}

		// load filestore
		for _, v := range exportedData.Data["filestore"] {
			logger.Infof("checksum: %s, data: %s", v.(map[string]interface{})["Chksum"], v.(map[string]interface{})["Data"])
			file_data, err := base64.StdEncoding.DecodeString(v.(map[string]interface{})["Data"].(string))
			if err != nil {
				return err
			}
			fd_buf := bytes.NewBuffer(file_data)
			fd_rc := ioutil.NopCloser(fd_buf)
			fs, err := filestore.New(v.(map[string]interface{})["Chksum"].(string), fd_rc, int64(fd_buf.Len()))
			if err != nil {
				return err
			}
			if err = fs.Save(); err != nil {
				return err
			}
		}

		// load cookbooks
		for _, v := range exportedData.Data["cookbook"] {
			if cb, err := cookbook.New(v.(map[string]interface{})["Name"].(string)); err != nil {
				return err
			} else {
				gerr := cb.Save()
				if gerr != nil {
					return gerr
				}
				for ver, cbv_data := range v.(map[string]interface{})["Versions"].(map[string]interface{}) {
					_, cbverr := cb.NewVersion(ver, cbv_data.(map[string]interface{}))
					if cbverr != nil {
						return cbverr
					}
				}
			}
		}

		// load data bags
		for _, v := range exportedData.Data["data_bag"] {
			if dbag, err := data_bag.New(v.(map[string]interface{})["Name"].(string)); err != nil {
				return err
			} else {
				gerr := dbag.Save()
				if gerr != nil {
					return gerr
				}
				for _, dbag_data := range v.(map[string]interface{})["DataBagItems"].(map[string]interface{}) {
					_, dbierr := dbag.NewDBItem(dbag_data.(map[string]interface{}))
					if dbierr != nil {
						return dbierr
					}
				}
				gerr = dbag.Save()
				if gerr != nil {
					return gerr
				}
			}
		}
		// load environments
		for _, v := range exportedData.Data["environment"] {
			if v.(map[string]interface{})["name"].(string) != "_default" {
				if e, err := environment.NewFromJson(v.(map[string]interface{})); err != nil {
					return err
				} else {
					gerr := e.Save()
					if gerr != nil {
						return gerr
					}
				}
			}
		}

	} else {
		err := fmt.Errorf("goiardi export data version %d.%d is not supported by this version of goiardi", exportedData.MajorVersion, exportedData.MinorVersion)
		return err
	}
	return nil
}
