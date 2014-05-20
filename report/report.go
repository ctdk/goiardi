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

/* Package report implements reporting on client runs and node changes. See 
http://docs.opscode.com/reporting.html for details. CURRENTLY EXPERIMENTAL. */
package report

import (
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/util"
	"github.com/ctdk/goiardi/data_store"
	"bytes"
	"encoding/gob"
	"time"
	"net/http"
	"strconv"
	"database/sql"
	"github.com/codeskyblue/go-uuid"
)

// The format for reporting start and end times in JSON. Of course, subtly
// different from MySQL's time format, but only subtly.
const ReportTimeFormat = "2006-01-02 15:04:05 -0700"

type Report struct {
	RunId string `json:"run_id"`
	StartTime time.Time `json:"start_time"`
	EndTime time.Time `json:"end_time"`
	TotalResCount int `json:"total_res_count"`
	Status string `json:"status"`
	RunList []string `json:"run_list"`
	Resources []map[string]interface{} `json:"resources"`
	Data map[string]interface{} `json:"data"` // I think this is right
	nodeName string
	organizationId int
}

type privReport struct {
	RunId *string
	StartTime *time.Time
	EndTime *time.Time
	TotalResCount *int
	Status *string 
	RunList *[]string
	Resources *[]map[string]interface{}
	Data *map[string]interface{}
	NodeName *string
	OrganizationId *int
}

func New(runId string, nodeName string) (*Report, util.Gerror) {
	var found bool
	if config.Config.UseMySQL {
		var err error
		found, err = checkForReportMySQL(data_store.Dbh, runId)
		if err != nil {
			gerr := util.CastErr(err)
			gerr.SetStatus(http.StatusInternalServerError)
			return nil, gerr
		}
	} else {
		ds := data_store.New()
		_, found = ds.Get("report", runId)
	}
	if found {
		err := util.Errorf("Report already exists")
		err.SetStatus(http.StatusConflict)
		return nil, err
	}
	if u := uuid.Parse(runId); u == nil {
		err := util.Errorf("run id was not a valid uuid")
		err.SetStatus(http.StatusBadRequest)
		return nil, err
	}
	report := &Report{
		RunId: runId,
		nodeName: nodeName,
	}
	return report, nil
}

func Get(runId string) (*Report, util.Gerror) {
	var report *Report
	var found bool
	if config.Config.UseMySQL {
		var err error
		report, err = getReportMySQL(runId)
		if err != nil {
			if err == sql.ErrNoRows {
				found = false
			} else {
				gerr := util.CastErr(err)
				gerr.SetStatus(http.StatusInternalServerError)
				return nil, gerr
			}
		} else {
			found = true 
		}
	} else {
		ds := data_store.New()
		var r interface{}
		r, found = ds.Get("report", runId)
		if r != nil {
			report = r.(*Report)
		}
	}
	if !found {
		err := util.Errorf("Report %s not found", runId)
		err.SetStatus(http.StatusNotFound)
		return nil, err
	}
	return report, nil
}

func (r *Report)Save() error {
	if config.Config.UseMySQL {
		return r.saveMySQL()
	} else {
		ds := data_store.New()
		ds.Set("report", r.RunId, r)
	}
	return nil
}

func (r *Report)Delete() error {
	if config.Config.UseMySQL {
		return r.deleteMySQL()
	} else {
		ds := data_store.New()
		ds.Delete("report", r.RunId)
	}
	return nil
}

func NewFromJson(node_name string, json_report map[string]interface{}) (*Report, util.Gerror) {
	rid, ok := json_report["run_id"].(string)
	if !ok {
		err := util.Errorf("invalid run id")
		err.SetStatus(http.StatusBadRequest)
		return nil, err
	}

	if action, ok := json_report["action"].(string); ok {
		if action != "start" {
			err := util.Errorf("invalid action %s", action)
			return nil, err
		}
	} else {
		err := util.Errorf("invalid action")
		return nil, err
	}
	stime, ok := json_report["start_time"].(string)
	if !ok {
		err := util.Errorf("invalid start time")
		return nil, err
	}
	start_time, terr := time.Parse(ReportTimeFormat, stime)
	if terr != nil {
		err := util.CastErr(terr)
		return nil, err
	}
	
	report, err := New(rid, node_name)
	if err != nil {
		return nil, err
	}
	report.StartTime = start_time
	if err != nil {
		return nil, err
	}
	return report, nil
}

func (r *Report)UpdateFromJson(json_report map[string]interface{}) util.Gerror {
	if action, ok := json_report["action"].(string); ok {
		if action != "end" {
			err := util.Errorf("invalid action %s", action)
			return err
		}
	} else {
		err := util.Errorf("invalid action")
		return err
	}
	etime, ok := json_report["end_time"].(string)
	if !ok {
		err := util.Errorf("invalid end time")
		return err
	}
	end_time, terr := time.Parse(ReportTimeFormat, etime)
	if terr != nil {
		err := util.CastErr(terr)
		return err
	}
	t, ok := json_report["total_res_count"].(string)
	if !ok {
		err := util.Errorf("invalid total_res_count")
		return err
	}
	trc, err := strconv.Atoi(t)
	if err != nil {
		err := util.Errorf("Error converting %v to int: %s", t, err.Error())
		return err
	}
	status, ok := json_report["status"].(string)
	if ok {
		if status != "success" && status != "failure" {
			err := util.Errorf("invalid status %s", status)
			return err
		}
	} else {
		err := util.Errorf("invalid status")
		return err
	}
	run_list, ok := json_report["run_list"].([]string)
	if !ok {
		err := util.Errorf("invalid run_list")
		return err
	}
	resources, ok := json_report["resources"].([]map[string]interface{})
	if !ok {
		err := util.Errorf("invalid resources")
		return err
	}
	data, ok := json_report["data"].(map[string]interface{})
	if !ok {
		err := util.Errorf("invalid data")
		return err
	}

	r.EndTime = end_time
	r.TotalResCount = trc
	r.Status = status
	r.RunList = run_list
	r.Resources = resources
	r.Data = data
	return nil
}

func GetList() []string {
	var report_list []string
	if config.Config.UseMySQL {

	} else {
		ds := data_store.New()
		report_list = ds.GetList("report")
	}
	return report_list
}

func GetReportList() ([]*Report, error) {
	if config.Config.UseMySQL {
		return getReportListMySQL()
	} else {
		reports := make([]*Report, 0)
		report_list := GetList()
		for _, r := range report_list {
			rp, _ := Get(r)
			if rp != nil {
				reports = append(reports, rp)
			}
		}
		return reports, nil
	}
}

func GetNodeList(nodeName string) ([]*Report, error) {
	if config.Config.UseMySQL {
		return getNodeListMySQL(nodeName)
	} else {
		// Really really not the most efficient way, but deliberately
		// not doing it in a better manner for now. If reporting
		// performance becomes a concern, SQL mode is probably a better
		// choice
		reports, _ := GetReportList()
		node_report_list := make([]*Report, 0)
		for _, r := range reports {
			if nodeName == r.nodeName {
				node_report_list = append(node_report_list, r)
			}
		}
		return node_report_list, nil
	}
}

func (r *Report) export() *privReport {
	return &privReport{ RunId: &r.RunId, StartTime: &r.StartTime, EndTime: &r.EndTime, TotalResCount: &r.TotalResCount, Status: &r.Status, Resources: &r.Resources, Data: &r.Data, NodeName: &r.nodeName, OrganizationId: &r.organizationId }
}

func (r *Report) GobEncode() ([]byte, error) {
	prv := r.export()
	buf := new(bytes.Buffer)
	decoder := gob.NewEncoder(buf)
	if err := decoder.Encode(prv); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (r *Report) GobDecode(b []byte) error {
	prv := r.export()
	buf := bytes.NewReader(b)
	encoder := gob.NewDecoder(buf)
	err := encoder.Decode(prv)
	if err != nil {
		return err
	}

	return nil
}
