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
	//"github.com/ctdk/goiardi/config"
	"encoding/json"
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
	"os"
	"time"
)

type ExportData struct {
	MajorVersion int
	MinorVersion int
	CreatedTime time.Time
	Data map[string][]interface{}
}

const ExportMajorVersion = 1
const ExportMinorVersion = 0

// Export all data to a json file. This can help with upgrading goiardi if save
// file compatibitity is broken between releases, or with transferring goiardi
// data between different backends.
func Export(fileName string) error {
	exportedData := &ExportData{ MajorVersion: ExportMajorVersion, MinorVersion: ExportMinorVersion, CreatedTime: time.Now() }
	exportedData.Data = make(map[string][]interface{})
	// ... and march through everything.
	exportedData.Data["clients"] = client.AllClients()
	exportedData.Data["cookbooks"] = cookbook.AllCookbooks()
	exportedData.Data["data_bag"] = data_bag.AllDataBags()
	exportedData.Data["environment"] = environment.AllEnvironments()
	exportedData.Data["filestore"] = filestore.AllFilestores()
	exportedData.Data["log_info"] = log_info.AllLogInfos()
	exportedData.Data["node"] = node.AllNodes()
	exportedData.Data["report"] = report.AllReports()
	exportedData.Data["role"] = role.AllRoles()
	exportedData.Data["sandbox"] = sandbox.AllSandboxes()
	exportedData.Data["user"] = user.AllUsers()

	fp, err := os.Create(fileName)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(fp)
	if err = enc.Encode(&exportedData); err != nil {
		return err
	}
	return nil
}

