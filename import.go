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
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/ctdk/goas/v2/logger"
	"github.com/ctdk/goiardi/client"
	"github.com/ctdk/goiardi/cookbook"
	"github.com/ctdk/goiardi/data_bag"
	"github.com/ctdk/goiardi/environment"
	"github.com/ctdk/goiardi/filestore"
	"github.com/ctdk/goiardi/log_info"
	"github.com/ctdk/goiardi/node"
	"github.com/ctdk/goiardi/report"
	"github.com/ctdk/goiardi/role"
	"github.com/ctdk/goiardi/sandbox"
	"github.com/ctdk/goiardi/user"
	"io/ioutil"
	"os"
	"time"
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

		// load clients
		logger.Infof("Loading clients")
		for _, v := range exportedData.Data["client"] {
			if c, err := client.NewFromJson(v.(map[string]interface{})); err != nil {
				return err
			} else {
				c.SetPublicKey(v.(map[string]interface{})["public_key"])
				gerr := c.Save()
				if gerr != nil {
					return gerr
				}
			}
		}

		// load users
		logger.Infof("Loading users")
		for _, v := range exportedData.Data["user"] {
			pwhash, _ := v.(map[string]interface{})["password"].(string)
			v.(map[string]interface{})["password"] = ""
			if u, err := user.NewFromJson(v.(map[string]interface{})); err != nil {
				return err
			} else {
				u.SetPasswdHash(pwhash)
				u.SetPublicKey(v.(map[string]interface{})["public_key"])
				gerr := u.Save()
				if gerr != nil {
					return gerr
				}
			}
		}

		// load filestore
		logger.Infof("Loading filestore")
		for _, v := range exportedData.Data["filestore"] {
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
		logger.Infof("Loading cookbooks")
		for _, v := range exportedData.Data["cookbook"] {
			if cb, err := cookbook.New(v.(map[string]interface{})["Name"].(string)); err != nil {
				return err
			} else {
				gerr := cb.Save()
				if gerr != nil {
					return gerr
				}
				for ver, cbv_data := range v.(map[string]interface{})["Versions"].(map[string]interface{}) {
					cbv_data, cerr := checkAttrs(cbv_data.(map[string]interface{}))
					if cerr != nil {
						return cerr
					}
					_, cbverr := cb.NewVersion(ver, cbv_data)
					if cbverr != nil {
						return cbverr
					}
				}
			}
		}

		// load data bags
		logger.Infof("Loading data bags")
		for _, v := range exportedData.Data["data_bag"] {
			if dbag, err := data_bag.New(v.(map[string]interface{})["Name"].(string)); err != nil {
				return err
			} else {
				gerr := dbag.Save()
				if gerr != nil {
					return gerr
				}
				for _, dbag_data := range v.(map[string]interface{})["DataBagItems"].(map[string]interface{}) {
					_, dbierr := dbag.NewDBItem(dbag_data.(map[string]interface{})["raw_data"].(map[string]interface{}))
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
		logger.Infof("Loading environments")
		for _, v := range exportedData.Data["environment"] {
			env_data, cerr := checkAttrs(v.(map[string]interface{}))
			if cerr != nil {
				return nil
			}
			if env_data["name"].(string) != "_default" {
				if e, err := environment.NewFromJson(env_data); err != nil {
					return err
				} else {
					gerr := e.Save()
					if gerr != nil {
						return gerr
					}
				}
			}
		}

		// load nodes
		logger.Infof("Loading nodes")
		for _, v := range exportedData.Data["node"] {
			node_data, cerr := checkAttrs(v.(map[string]interface{}))
			if cerr != nil {
				return nil
			}
			if n, err := node.NewFromJson(node_data); err != nil {
				return err
			} else {
				gerr := n.Save()
				if gerr != nil {
					return gerr
				}
			}
		}

		// load roles
		logger.Infof("Loading roles")
		for _, v := range exportedData.Data["role"] {
			role_data, cerr := checkAttrs(v.(map[string]interface{}))
			if cerr != nil {
				return nil
			}
			if r, err := role.NewFromJson(role_data); err != nil {
				return err
			} else {
				gerr := r.Save()
				if gerr != nil {
					return gerr
				}
			}
		}

		// load sandboxes
		logger.Infof("Loading sandboxes")
		for _, v := range exportedData.Data["sandbox"] {
			sbid, _ := v.(map[string]interface{})["Id"].(string)
			sbts, _ := v.(map[string]interface{})["CreationTime"].(string)
			sbcomplete, _ := v.(map[string]interface{})["Completed"].(bool)
			sbck, _ := v.(map[string]interface{})["Checksums"].([]interface{})
			sbTime, err := time.Parse(time.RFC3339, sbts)
			if err != nil {
				return err
			}
			sbChecksums := make([]string, len(sbck))
			for i, c := range sbck {
				sbChecksums[i] = c.(string)
			}
			sbox := &sandbox.Sandbox{Id: sbid, CreationTime: sbTime, Completed: sbcomplete, Checksums: sbChecksums}
			if err = sbox.Save(); err != nil {
				return err
			}
		}

		// load log_infos
		logger.Infof("Loading log_info")
		for _, v := range exportedData.Data["log_info"] {
			if err := log_info.Import(v.(map[string]interface{})); err != nil {
				return err
			}
		}

		// load reports
		logger.Infof("Loading reports")
		for _, v := range exportedData.Data["report"] {
			nodeName := v.(map[string]interface{})["node_name"].(string)
			v.(map[string]interface{})["action"] = "start"
			if st, ok := v.(map[string]interface{})["start_time"].(string); ok {
				t, err := time.Parse(time.RFC3339, st)
				if err != nil {
					return err
				}
				v.(map[string]interface{})["start_time"] = t.Format(report.ReportTimeFormat)
			}
			if et, ok := v.(map[string]interface{})["end_time"].(string); ok {
				t, err := time.Parse(time.RFC3339, et)
				if err != nil {
					return err
				}
				v.(map[string]interface{})["end_time"] = t.Format(report.ReportTimeFormat)
			}
			if r, err := report.NewFromJson(nodeName, v.(map[string]interface{})); err != nil {
				return err
			} else {
				gerr := r.Save()
				if gerr != nil {
					return gerr
				}
				v.(map[string]interface{})["action"] = "end"
				if err := r.UpdateFromJson(v.(map[string]interface{})); err != nil {
					return err
				} else {
					gerr := r.Save()
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
