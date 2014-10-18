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

/*
Package report implements reporting on client runs and node changes. See http://docs.opscode.com/reporting.html for details. CURRENTLY EXPERIMENTAL. */
package report

import (
	"bytes"
	"database/sql"
	"encoding/gob"
	"github.com/codeskyblue/go-uuid"
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/datastore"
	"github.com/ctdk/goiardi/organization"
	"github.com/ctdk/goiardi/util"
	"net/http"
	"strconv"
	"time"
)

// The format for reporting start and end times in JSON. Of course, subtly
// different from MySQL's time format, but only subtly.
const ReportTimeFormat = "2006-01-02 15:04:05 -0700"

// Report holds information on a chef client's run, including when, what
// resources changed, what recipes were in the run list, and whether the run was
// successful or not.
type Report struct {
	RunID          string                 `json:"run_id"`
	StartTime      time.Time              `json:"start_time"`
	EndTime        time.Time              `json:"end_time"`
	TotalResCount  int                    `json:"total_res_count"`
	Status         string                 `json:"status"`
	RunList        string                 `json:"run_list"`
	Resources      []interface{}          `json:"resources"`
	Data           map[string]interface{} `json:"data"` // I think this is right
	NodeName       string                 `json:"nodeName"`
	// might be able to remove this
	organizationID int
	org *organization.Organization
}

type privReport struct {
	RunID          *string
	StartTime      *time.Time
	EndTime        *time.Time
	TotalResCount  *int
	Status         *string
	RunList        *string
	Resources      *[]interface{}
	Data           *map[string]interface{}
	NodeName       *string
	OrganizationID *int
}

// New creates a new report.
func New(org *organization.Organization, runID string, nodeName string) (*Report, util.Gerror) {
	var found bool
	if config.UsingDB() {
		var err error
		found, err = checkForReportSQL(datastore.Dbh, runID)
		if err != nil {
			gerr := util.CastErr(err)
			gerr.SetStatus(http.StatusInternalServerError)
			return nil, gerr
		}
	} else {
		ds := datastore.New()
		_, found = ds.Get(org.DataKey("report"), runID)
	}
	if found {
		err := util.Errorf("Report already exists")
		err.SetStatus(http.StatusConflict)
		return nil, err
	}
	if u := uuid.Parse(runID); u == nil {
		err := util.Errorf("run id was not a valid uuid")
		err.SetStatus(http.StatusBadRequest)
		return nil, err
	}
	report := &Report{
		RunID:    runID,
		NodeName: nodeName,
		Status:   "started",
		org: org,
	}
	return report, nil
}

// Get a report.
func Get(org *organization.Organization, runID string) (*Report, util.Gerror) {
	var report *Report
	var found bool
	if config.UsingDB() {
		var err error
		report, err = getReportSQL(runID)
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
		ds := datastore.New()
		var r interface{}
		r, found = ds.Get(org.DataKey("report"), runID)
		if r != nil {
			report = r.(*Report)
			report.org = org
		}
	}
	if !found {
		err := util.Errorf("Report %s not found", runID)
		err.SetStatus(http.StatusNotFound)
		return nil, err
	}
	return report, nil
}

// Save a report.
func (r *Report) Save() error {
	if config.Config.UseMySQL {
		return r.saveMySQL()
	} else if config.Config.UsePostgreSQL {
		return r.savePostgreSQL()
	} else {
		ds := datastore.New()
		ds.Set(r.org.DataKey("report"), r.RunID, r)
	}
	return nil
}

// Delete a report.
func (r *Report) Delete() error {
	if config.UsingDB() {
		return r.deleteSQL()
	}
	ds := datastore.New()
	ds.Delete(r.org.DataKey("report"), r.RunID)
	return nil
}

// NewFromJSON creates a new report from the given uploaded JSON.
func NewFromJSON(org *organization.Organization, nodeName string, jsonReport map[string]interface{}) (*Report, util.Gerror) {
	rid, ok := jsonReport["run_id"].(string)
	if !ok {
		err := util.Errorf("invalid run id")
		err.SetStatus(http.StatusBadRequest)
		return nil, err
	}

	if action, ok := jsonReport["action"].(string); ok {
		if action != "start" {
			err := util.Errorf("invalid action %s", action)
			return nil, err
		}
	} else {
		err := util.Errorf("invalid action")
		return nil, err
	}
	stime, ok := jsonReport["start_time"].(string)
	if !ok {
		err := util.Errorf("invalid start time")
		return nil, err
	}
	startTime, terr := time.Parse(ReportTimeFormat, stime)
	if terr != nil {
		err := util.CastErr(terr)
		return nil, err
	}

	report, err := New(org, rid, nodeName)
	if err != nil {
		return nil, err
	}
	report.StartTime = startTime
	if err != nil {
		return nil, err
	}
	return report, nil
}

// UpdateFromJSON updates a report with the values in the uploaded JSON.
func (r *Report) UpdateFromJSON(jsonReport map[string]interface{}) util.Gerror {
	if action, ok := jsonReport["action"].(string); ok {
		if action != "end" {
			err := util.Errorf("invalid action %s", action)
			return err
		}
	} else {
		err := util.Errorf("invalid action")
		return err
	}
	_, ok := jsonReport["end_time"].(string)
	if !ok {
		err := util.Errorf("invalid end time")
		return err
	}
	endTime, terr := time.Parse(ReportTimeFormat, jsonReport["end_time"].(string))
	if terr != nil {
		err := util.CastErr(terr)
		return err
	}
	var trc int
	switch t := jsonReport["total_res_count"].(type) {
	case string:
		var err error
		trc, err = strconv.Atoi(t)
		if err != nil {
			err := util.Errorf("Error converting %v to int: %s", jsonReport["total_res_count"], err.Error())
			return err
		}
	case float64:
		trc = int(t)
	case int:
		trc = t
	default:
		err := util.Errorf("invalid total_res_count %T", t)
		return err
	}
	status, ok := jsonReport["status"].(string)
	if ok {
		if status != "success" && status != "failure" {
			err := util.Errorf("invalid status %s", status)
			return err
		}
	} else {
		err := util.Errorf("invalid status")
		return err
	}
	_, ok = jsonReport["run_list"].(string)
	if !ok {
		err := util.Errorf("invalid run_list")
		return err
	}
	_, ok = jsonReport["resources"].([]interface{})
	if !ok {
		err := util.Errorf("invalid resources %T", jsonReport["resources"])
		return err
	}
	_, ok = jsonReport["data"].(map[string]interface{})
	if !ok {
		err := util.Errorf("invalid data")
		return err
	}

	r.EndTime = endTime
	r.TotalResCount = trc
	r.Status = jsonReport["status"].(string)
	r.RunList = jsonReport["run_list"].(string)
	r.Resources = jsonReport["resources"].([]interface{})
	r.Data = jsonReport["data"].(map[string]interface{})
	return nil
}

// GetList returns a list of UUIDs of reports on the system.
func GetList(org *organization.Organization) []string {
	var reportList []string
	if config.UsingDB() {
		reportList = getListSQL()
	} else {
		ds := datastore.New()
		reportList = ds.GetList(org.DataKey("report"))
	}
	return reportList
}

// GetReportList returns a list of reports on the system in the given time range
// and with the given status, which may be "" for any status.
func GetReportList(org *organization.Organization, from, until time.Time, rows int, status string) ([]*Report, error) {
	if config.UsingDB() {
		return getReportListSQL(from, until, rows, status)
	}
	var reports []*Report
	reportList := GetList(org)
	i := 0
	for _, r := range reportList {
		rp, _ := Get(org, r)
		if rp != nil && rp.checkTimeRange(from, until) && (status == "" || (status != "" && rp.Status == status)) {
			reports = append(reports, rp)
			i++
		}
		if i > rows {
			break
		}
	}
	return reports, nil
}

func (r *Report) checkTimeRange(from, until time.Time) bool {
	return r.StartTime.After(from) && r.StartTime.Before(until)
}

// GetNodeList returns a list of reports from the given node in the time range
// and status given. Status may be "" for all statuses.
func GetNodeList(org *organization.Organization, nodeName string, from, until time.Time, rows int, status string) ([]*Report, error) {
	if config.UsingDB() {
		return getNodeListSQL(nodeName, from, until, rows, status)
	}
	// Really really not the most efficient way, but deliberately
	// not doing it in a better manner for now. If reporting
	// performance becomes a concern, SQL mode is probably a better
	// choice
	reports, _ := GetReportList(org, from, until, rows, status)
	var nodeReportList []*Report
	for _, r := range reports {
		if nodeName == r.NodeName && (status == "" || (status != "" && r.Status == status)) {
			nodeReportList = append(nodeReportList, r)
		}
	}
	return nodeReportList, nil
}

func (r *Report) export() *privReport {
	return &privReport{RunID: &r.RunID, StartTime: &r.StartTime, EndTime: &r.EndTime, TotalResCount: &r.TotalResCount, Status: &r.Status, Resources: &r.Resources, Data: &r.Data, NodeName: &r.NodeName, OrganizationID: &r.organizationID}
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

// AllReports returns all run reports currently on the server for export.
func AllReports(org *organization.Organization) []*Report {
	if config.UsingDB() {
		return getReportsSQL()
	}
	//var reports []*Report
	reportList := GetList(org)
	reports := make([]*Report, 0, len(reportList))
	for _, r := range reportList {
		rp, _ := Get(org, r)
		if rp != nil {
			reports = append(reports, rp)
		}
	}
	return reports
}
