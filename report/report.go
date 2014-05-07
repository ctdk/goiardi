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
	"time"
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
