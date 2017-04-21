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

package main

// TODO: This needs org support

import (
	"encoding/json"
	"fmt"
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
	"os"
	"time"
)

// ExportData is a struct describing the container holding the exported data (or
// the data file to import. It contains the major and minor version numbers, the
// time the dump was created, and a map holding all of the data objects.
type ExportData struct {
	MajorVersion int
	MinorVersion int
	CreatedTime  time.Time
	// It's a map of interfaces because the object structs may change
	// between releases.
	Data interface{}
}

// Major version number of the export file format.
const ExportMajorVersion = 2

// Minor version number of the export file format.
const ExportMinorVersion = 0

// Export all data to a json file. This can help with upgrading goiardi if save
// file compatibitity is broken between releases, or with transferring goiardi
// data between different backends.

func exportAll(fileName string) error {
	exportedData := &ExportData{MajorVersion: ExportMajorVersion, MinorVersion: ExportMinorVersion, CreatedTime: time.Now()}
	orgs := organization.AllOrganizations()
	AllData := make(map[string]interface{})
	AllData["org_objects"] = make(map[string]map[string][]interface{})
	for _, org := range orgs {
		data := make(map[string][]interface{})
		// ... and march through everything.
		data["client"] = client.ExportAllClients(org)
		data["cookbook"] = exportTransformSlice(cookbook.AllCookbooks(org))
		data["databag"] = exportTransformSlice(databag.AllDataBags(org))
		data["environment"] = exportTransformSlice(environment.AllEnvironments(org))
		data["filestore"] = exportTransformSlice(filestore.AllFilestores(org.Name))
		data["loginfo"] = exportTransformSlice(loginfo.AllLogInfos(org))
		data["node"] = exportTransformSlice(node.AllNodes(org))
		data["node_status"] = exportTransformSlice(node.AllNodeStatuses(org))
		data["report"] = exportTransformSlice(report.AllReports(org))
		data["role"] = exportTransformSlice(role.AllRoles(org))
		data["sandbox"] = exportTransformSlice(sandbox.AllSandboxes(org))
		data["shovey"] = exportTransformSlice(shovey.AllShoveys(org))
		data["shovey_run"] = exportTransformSlice(shovey.AllShoveyRuns(org))
		data["shovey_run_stream"] = exportTransformSlice(shovey.AllShoveyRunStreams(org))
		AllData["org_objects"].(map[string]map[string][]interface{})[org.Name] = data
	}
	AllData["organization"] = organization.ExportAllOrgs()
	AllData["user"] = user.ExportAllUsers()
	exportedData.Data = AllData

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
	case []*databag.DataBag:
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
	case []*loginfo.LogInfo:
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
	case []*node.NodeStatus:
		exp = make([]interface{}, len(data))
		for i, v := range data {
			exp[i] = v
		}
	case []*shovey.Shovey:
		exp = make([]interface{}, len(data))
		for i, v := range data {
			exp[i] = v
		}
	case []*shovey.ShoveyRun:
		exp = make([]interface{}, len(data))
		for i, v := range data {
			exp[i] = v
		}
	case []*shovey.ShoveyRunStream:
		exp = make([]interface{}, len(data))
		for i, v := range data {
			exp[i] = v
		}
	case []*organization.Organization:
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
