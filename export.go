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
	"fmt"
)

type ExportData struct {
	MajorVersion int
	MinorVersion int
	CreatedTime time.Time
	// It's a map of interfaces because the object structs may change
	// between releases.
	Data map[string][]interface{}
}

const ExportMajorVersion = 1
const ExportMinorVersion = 0

// Export all data to a json file. This can help with upgrading goiardi if save
// file compatibitity is broken between releases, or with transferring goiardi
// data between different backends.

func exportAll(fileName string) error {
	exportedData := &ExportData{ MajorVersion: ExportMajorVersion, MinorVersion: ExportMinorVersion, CreatedTime: time.Now() }
	exportedData.Data = make(map[string][]interface{})
	// ... and march through everything.
	exportedData.Data["clients"] = client.ExportAllClients()
	exportedData.Data["cookbooks"] = exportTransformSlice(cookbook.AllCookbooks())
	exportedData.Data["data_bag"] = exportTransformSlice(data_bag.AllDataBags())
	exportedData.Data["environment"] = exportTransformSlice(environment.AllEnvironments())
	exportedData.Data["filestore"] = exportTransformSlice(filestore.AllFilestores())
	exportedData.Data["log_info"] = exportTransformSlice(log_info.AllLogInfos())
	exportedData.Data["node"] = exportTransformSlice(node.AllNodes())
	exportedData.Data["report"] = exportTransformSlice(report.AllReports())
	exportedData.Data["role"] = exportTransformSlice(role.AllRoles())
	exportedData.Data["sandbox"] = exportTransformSlice(sandbox.AllSandboxes())
	exportedData.Data["user"] = user.ExportAllUsers()

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

func exportTransformSlice(data interface{}) []interface{} {
	var exp []interface{}
	switch data := data.(type) {
		case []*client.Client:
			exp = make([]interface{}, len(data))
			for i, v := range data {
				exp[i] = v
			}
		case []*user.User:
			exp = make([]interface{}, len(data))
			for i, v := range data {
				exp[i] = v
			}
		case []*cookbook.Cookbook:
			exp = make([]interface{}, len(data))
			for i, v := range data {
				exp[i] = v
			}
		case []*data_bag.DataBag:
			exp = make([]interface{}, len(data))
			for i, v := range data {
				exp[i] = v
			}
		case []*environment.ChefEnvironment:
			exp = make([]interface{}, len(data))
			for i, v := range data {
				exp[i] = v
			}
		case []*filestore.FileStore:
			exp = make([]interface{}, len(data))
			for i, v := range data {
				exp[i] = v
			}
		case []*log_info.LogInfo:
			exp = make([]interface{}, len(data))
			for i, v := range data {
				exp[i] = v
			}
		case []*node.Node:
			exp = make([]interface{}, len(data))
			for i, v := range data {
				exp[i] = v
			}
		case []*report.Report:
			exp = make([]interface{}, len(data))
			for i, v := range data {
				exp[i] = v
			}
		case []*role.Role:
			exp = make([]interface{}, len(data))
			for i, v := range data {
				exp[i] = v
			}
		case []*sandbox.Sandbox:
			exp = make([]interface{}, len(data))
			for i, v := range data {
				exp[i] = v
			}
		default:
			msg := fmt.Sprintf("Type %t was passed in, but that isn't handled with export.", data)
			panic(msg)
	}
	return exp
}
