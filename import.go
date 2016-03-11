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

// TODO: This will need org support

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/ctdk/goiardi/client"
	"github.com/ctdk/goiardi/cookbook"
	"github.com/ctdk/goiardi/databag"
	"github.com/ctdk/goiardi/environment"
	"github.com/ctdk/goiardi/filestore"
	"github.com/ctdk/goiardi/loginfo"
	"github.com/ctdk/goiardi/node"
	"github.com/ctdk/goiardi/organization"
	"github.com/ctdk/goiardi/report"
	"github.com/ctdk/goiardi/role"
	"github.com/ctdk/goiardi/sandbox"
	"github.com/ctdk/goiardi/shovey"
	"github.com/ctdk/goiardi/user"
	"github.com/tideland/golib/logger"
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
	// TODO: version 2.0
	var orgs []*organization.Organization
	if exportedData.MajorVersion == 1 {
		org, err := organization.Get("default")
		if err != nil {
			return err
		}
		orgs = []*organization.Organization{org}
	}
	//if exportedData.MajorVersion == 1 && (exportedData.MinorVersion == 0 || exportedData.MinorVersion == 1) {
	if exportedData.MajorVersion <= 2 {
		logger.Infof("Importing data, version %d.%d created on %s", exportedData.MajorVersion, exportedData.MinorVersion, exportedData.CreatedTime)

		if exportedData.MajorVersion == 2 {
			for _, v := range exportedData.Data.(map[string]interface{})["organization"].([]interface{}) {
				if org, err := organization.Import(v.(map[string]interface{})); err != nil {
					return err
				} else {
					orgs = append(orgs, org)
				}
			}
		}

		// load users
		logger.Infof("Loading users")
		var userData []interface{}
		if exportedData.MajorVersion == 2 {
			userData = exportedData.Data.(map[string]interface{})["user"].([]interface{})
		} else {
			userData = exportedData.Data.(map[string][]interface{})["user"]
		}
		for _, v := range userData {
			pwhash, _ := v.(map[string]interface{})["password"].(string)
			v.(map[string]interface{})["password"] = ""
			u, err := user.NewFromJSON(v.(map[string]interface{}))
			if err != nil {
				return err
			}
			u.SetPasswdHash(pwhash)
			pkerr := u.SetPublicKey(v.(map[string]interface{})["public_key"])
			if pkerr != nil {
				return pkerr
			}
			gerr := u.Save()
			if gerr != nil {
				return gerr
			}
		}
		// TODO: perm/acl/etc. will need to be dealt with.

		for _, org := range orgs {
			var data map[string][]interface{}
			if exportedData.MajorVersion == 2 {
				data = exportedData.Data.(map[string]interface{})["org_objects"].(map[string]map[string][]interface{})[org.Name]
			} else {
				data = exportedData.Data.(map[string][]interface{})
			}

			// load clients
			logger.Infof("Loading clients")
			for _, v := range data["client"] {
				c, err := client.NewFromJSON(org, v.(map[string]interface{}))
				if err != nil {
					return err
				}
				c.SetPublicKey(v.(map[string]interface{})["public_key"])
				gerr := c.Save()
				if gerr != nil {
					return gerr
				}
			}

			// load filestore
			logger.Infof("Loading filestore")
			for _, v := range data["filestore"] {
				fileData, err := base64.StdEncoding.DecodeString(v.(map[string]interface{})["Data"].(string))
				if err != nil {
					return err
				}
				fdBuf := bytes.NewBuffer(fileData)
				fdRc := ioutil.NopCloser(fdBuf)
				fs, err := filestore.New(org.Name, v.(map[string]interface{})["Chksum"].(string), fdRc, int64(fdBuf.Len()))
				if err != nil {
					return err
				}
				if err = fs.Save(); err != nil {
					return err
				}
			}

			// load cookbooks
			logger.Infof("Loading cookbooks")
			for _, v := range data["cookbook"] {
				cb, err := cookbook.New(org, v.(map[string]interface{})["Name"].(string))
				if err != nil {
					return err
				}
				gerr := cb.Save()
				if gerr != nil {
					return gerr
				}
				for ver, cbvData := range v.(map[string]interface{})["Versions"].(map[string]interface{}) {
					cbvData, cerr := checkAttrs(cbvData.(map[string]interface{}))
					if cerr != nil {
						return cerr
					}
					_, cbverr := cb.NewVersion(ver, cbvData)
					if cbverr != nil {
						return cbverr
					}
				}
			}

			// load data bags
			logger.Infof("Loading data bags")
			for _, v := range data["data_bag"] {
				dbag, err := databag.New(org, v.(map[string]interface{})["Name"].(string))
				if err != nil {
					return err
				}
				gerr := dbag.Save()
				if gerr != nil {
					return gerr
				}
				for _, dbagData := range v.(map[string]interface{})["DataBagItems"].(map[string]interface{}) {
					_, dbierr := dbag.NewDBItem(dbagData.(map[string]interface{})["raw_data"].(map[string]interface{}))
					if dbierr != nil {
						return dbierr
					}
				}
				gerr = dbag.Save()
				if gerr != nil {
					return gerr
				}
			}
			// load environments
			logger.Infof("Loading environments")
			for _, v := range data["environment"] {
				envData, cerr := checkAttrs(v.(map[string]interface{}))
				if cerr != nil {
					return nil
				}
				if envData["name"].(string) != "_default" {
					e, err := environment.NewFromJSON(org, envData)
					if err != nil {
						return err
					}
					gerr := e.Save()
					if gerr != nil {
						return gerr
					}
				}
			}

			// load nodes
			logger.Infof("Loading nodes")
			for _, v := range data["node"] {
				nodeData, cerr := checkAttrs(v.(map[string]interface{}))
				if cerr != nil {
					return nil
				}
				n, err := node.NewFromJSON(org, nodeData)
				if err != nil {
					return err
				}
				gerr := n.Save()
				if gerr != nil {
					return gerr
				}
			}

			// load roles
			logger.Infof("Loading roles")
			for _, v := range data["role"] {
				roleData, cerr := checkAttrs(v.(map[string]interface{}))
				if cerr != nil {
					return nil
				}
				r, err := role.NewFromJSON(org, roleData)
				if err != nil {
					return err
				}
				gerr := r.Save()
				if gerr != nil {
					return gerr
				}
			}

			// load reports
			logger.Infof("Loading reports")
			for _, o := range data["report"] {
				// handle data exported from a bugged report export
				var nodeName string
				v := o.(map[string]interface{})
				if n, ok := v["node_name"]; ok {
					nodeName = n.(string)
				} else if n, ok := v["nodeName"]; ok {
					nodeName = n.(string)
				}
				v["action"] = "start"
				if st, ok := v["start_time"].(string); ok {
					t, err := time.Parse(time.RFC3339, st)
					if err != nil {
						return err
					}
					v["start_time"] = t.Format(report.ReportTimeFormat)
				}
				if et, ok := v["end_time"].(string); ok {
					t, err := time.Parse(time.RFC3339, et)
					if err != nil {
						return err
					}
					v["end_time"] = t.Format(report.ReportTimeFormat)
				}
				r, err := report.NewFromJSON(org, nodeName, v)
				if err != nil {
					return err
				}
				gerr := r.Save()
				if gerr != nil {
					return gerr
				}
				v["action"] = "end"
				if err := r.UpdateFromJSON(v); err != nil {
					return err
				}
				gerr = r.Save()
				if gerr != nil {
					return gerr
				}
			}

			// load sandboxes
			logger.Infof("Loading sandboxes")
			for _, v := range data["sandbox"] {
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
				sbox := &sandbox.Sandbox{ID: sbid, CreationTime: sbTime, Completed: sbcomplete, Checksums: sbChecksums}
				if err = sbox.Save(); err != nil {
					return err
				}
			}

			// load loginfos
			logger.Infof("Loading loginfo")
			for _, v := range data["loginfo"] {
				if err := loginfo.Import(org, v.(map[string]interface{})); err != nil {
					return err
				}
			}

			// version 1.1 or greater
			if (exportedData.MajorVersion == 1 && exportedData.MinorVersion == 1) || exportedData.MajorVersion >= 2 {
				// import shovey jobs, run, and streams, and node
				// statuses
				logger.Infof("Loading node statuses...")
				for _, v := range data["node_status"] {
					ns := v.(map[string]interface{})
					err := node.ImportStatus(org, ns)
					if err != nil {
						return err
					}
				}
				logger.Infof("Loading shoveys...")
				for _, v := range data["shovey"] {
					s := v.(map[string]interface{})
					err := shovey.ImportShovey(org, s)
					if err != nil {
						return err
					}
				}
				logger.Infof("Loading shovey runs...")
				for _, v := range data["shovey_run"] {
					s := v.(map[string]interface{})
					err := shovey.ImportShoveyRun(org, s)
					if err != nil {
						return err
					}

				}
				logger.Infof("Loading shovey run streams...")
				for _, v := range data["shovey_run_stream"] {
					s := v.(map[string]interface{})
					err := shovey.ImportShoveyRunStream(org, s)
					if err != nil {
						return err
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
