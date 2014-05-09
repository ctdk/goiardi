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
http://docs.opscode.com/reporting.html for details. */
package report

import (
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/util"
	"github.com/ctdk/goiardi/data_store"
	"time"
	"net/http"
	"database/sql"
	"github.com/codeskyblue/go-uuid"
)

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
	var found
	if config.Config.UseMySQL {

	} else {
		ds := data_store.New()
		_, found = ds.Get("report", runId)
	}
	if found {
		err = util.Errorf("Report already exists")
		err.SetStatus(http.StatusConflict)
		return nil, err
	}
	report := &Report{
		RunId: runId,
		nodeName: nodeName
	}
	return report, nil
}

func Get(runId string) (*Report, util.Gerror) {
	var report *Report
	if config.Config.UseMySQL {

	} else {
		ds := data_store.New()
		r, found := ds.Get("report", runId)
		if !found {
			err := util.Errorf("Report %s not found", runId)
			err.SetStatus(http.StatusNotFound)
			return nil, err
		}
		if c != nil {
			report = r.(*Report)
		}
	}
	return report, nil
}

func (r *Report)Save() error {
	if config.Config.UseMySQL {

	} else {
		ds := data_store.New()
		ds.Set("report", r.RunId, r)
	}
	return nil
}

func (r *Report)Delete() error {
	if config.Config.UseMySQL {

	} else {
		ds := data_store.New()
		ds.Delete("report", r.RunId)
	}
	return nil
}

func NewFromJson(json_report map[string]interface{}) (*Report, util.Gerror) {
	rid, ok := json_report["run_id"].(string)
	if !ok {
		err := util.Errorf("invalid run id")
		err.SetStatus(http.StatusBadRequest)
		return nil, err
	}
	if u := uuid.Parse(rid); u == nil {
		err := util.Errorf("run id was not a valid uuid")
		err.SetStatus(http.StatusBadRequest)
		return nil, err
	}
	report, err := New(rid)
	if err != nil {
		return nil, err
	}
	err = report.UpdateFromJson(json_report)
	if err != nil {
		return nil, err
	}
	return report, nil
}

func (r *Report)UpdateFromJson(json_report map[string]interface{}) util.Gerror {

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

func GetReportList() []*Report {
	if config.Config.UseMySQL {
		return nil
	} else {
		reports := make([]*Report, 0)
		report_list := GetList()
		for _, r := range report_list {
			rp, _ := Get(r)
			if rp != nil {
				reports = append(reports, rp)
			}
		}
		return reports
	}
}

func GetNodeList(n *node.Node) []*Report {
	if config.Config.UseMySQL {
		return nil
	} else {
		// Really really not the most efficient way, but deliberately
		// not doing it in a better manner for now. If reporting
		// performance becomes a concern, SQL mode is probably a better
		// choice
		reports := GetReportList()
		node_report_list := make([]*Report, 0)
		for _, r := range reports {
			if n.Name == r.nodeName {
				node_report_list = append(node_report_list, r)
			}
		}
		return node_report_list
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
