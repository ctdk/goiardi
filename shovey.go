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
	"fmt"
	"github.com/ctdk/goas/v2/logger"
	"github.com/ctdk/goiardi/actor"
	"github.com/ctdk/goiardi/shovey"
	"github.com/ctdk/goiardi/util"
	"net/http"
	"strconv"
)

func shoveyHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	opUser, oerr := actor.GetReqUser(r.Header.Get("X-OPS-USERID"))
	if oerr != nil {
		jsonErrorReport(w, r, oerr.Error(), oerr.Status())
		return
	}
	if !opUser.IsAdmin() && r.Method != "PUT" {
		jsonErrorReport(w, r, "you cannot perform this action", http.StatusForbidden)
		return
	}
	pathArray := splitPath(r.URL.Path)
	pathArrayLen := len(pathArray)

	if pathArrayLen < 2 || pathArrayLen > 4 || pathArray[1] == "" {
		jsonErrorReport(w, r, "Bad request", http.StatusBadRequest)
		return
	}
	op := pathArray[1]

	shoveyResponse := make(map[string]interface{})

	switch op {
	case "jobs":
		switch r.Method {
		case "GET":
			switch pathArrayLen {
			case 4:
				shove, err := shovey.Get(pathArray[2])
				if err != nil {
					jsonErrorReport(w, r, err.Error(), err.Status())
					return
				}
				sj, err := shove.GetRun(pathArray[3])
				if err != nil {
					jsonErrorReport(w, r, err.Error(), err.Status())
					return
				}
				shoveyResponse, err = sj.ToJSON()
				if err != nil {
					jsonErrorReport(w, r, err.Error(), err.Status())
					return
				}
			case 3:
				shove, err := shovey.Get(pathArray[2])
				if err != nil {
					jsonErrorReport(w, r, err.Error(), err.Status())
					return
				}
				shoveyResponse, err = shove.ToJSON()
				if err != nil {
					jsonErrorReport(w, r, err.Error(), err.Status())
					return
				}
			default:
				shoveyIDs, err := shovey.AllShoveyIDs()
				if err != nil {
					jsonErrorReport(w, r, err.Error(), err.Status())
					return
				}
				enc := json.NewEncoder(w)
				if jerr := enc.Encode(&shoveyIDs); err != nil {
					jsonErrorReport(w, r, jerr.Error(), http.StatusInternalServerError)
				}
				return
			}
		case "POST":
			if pathArrayLen != 2 {
				jsonErrorReport(w, r, "Bad request", http.StatusBadRequest)
				return
			}
			shvData, err := parseObjJSON(r.Body)
			if err != nil {
				jsonErrorReport(w, r, err.Error(), http.StatusBadRequest)
				return
			}
			logger.Debugf("shvData: %v", shvData)
			var quorum string
			var timeout int
			var ok bool
			if quorum, ok = shvData["quorum"].(string); !ok {
				quorum = "100%"
			}
			logger.Debugf("run_timeout is a %T", shvData["run_timeout"])
			if t, ok := shvData["run_timeout"].(float64); !ok {
				timeout = 300
			} else {
				timeout = int(t)
			}
			if len(shvData["nodes"].([]interface{})) == 0 {
				jsonErrorReport(w, r, "no nodes provided", http.StatusBadRequest)
				return
			} 
			nodeNames := make([]string, len(shvData["nodes"].([]interface{})))
			for i, v := range shvData["nodes"].([]interface{}) {
				nodeNames[i] = v.(string)
			}
			
			s, gerr := shovey.New(shvData["command"].(string), timeout, quorum, nodeNames)
			if gerr != nil {
				jsonErrorReport(w, r, gerr.Error(), gerr.Status())
				return
			}
			gerr = s.Start()
			if gerr != nil {
				jsonErrorReport(w, r, gerr.Error(), gerr.Status())
				return
			}

			shoveyResponse["id"] = s.RunID
			shoveyResponse["uri"] = util.CustomURL(fmt.Sprintf("/shovey/jobs/%s", s.RunID))
		case "PUT":
			switch pathArrayLen {
			case 3:
				if pathArray[2] != "cancel" {
					jsonErrorReport(w, r, "Bad request", http.StatusBadRequest)
					return
				}
				cancelData, perr := parseObjJSON(r.Body)
				if perr != nil {
					jsonErrorReport(w, r, perr.Error(), http.StatusBadRequest)
					return
				}

				var nodeNames []string
				runID, ok := cancelData["run_id"].(string)
				if !ok {
					jsonErrorReport(w, r, "No shovey run ID provided, or provided id was invalid", http.StatusBadRequest)
					return
				}
				
				if nn, ok := cancelData["nodes"].([]interface{}); ok {
					for _, v := range nn {
						nodeNames = append(nodeNames, v.(string))
					}
				}
				shove, err := shovey.Get(runID)
				if err != nil {
					jsonErrorReport(w, r, err.Error(), err.Status())
					return
				}
				if len(nodeNames) != 0 {
					serr := shove.CancelRuns(nodeNames)
					if serr != nil {
						logger.Debugf("Error cancelling runs: %s", serr.Error())
						jsonErrorReport(w, r, err.Error(), err.Status())
						return
					}
				} else {
					err = shove.Cancel()
					if err != nil {
						jsonErrorReport(w, r, err.Error(), err.Status())
						return
					}
				}
				shoveyResponse, err = shove.ToJSON()
				if err != nil {
					jsonErrorReport(w, r, err.Error(), err.Status())
					return
				}
			case 4:
				sjData, perr := parseObjJSON(r.Body)
				if perr != nil {
					jsonErrorReport(w, r, perr.Error(), http.StatusBadRequest)
					return
				}
				nodeName := pathArray[3]
				logger.Debugf("sjData: %v", sjData)
				shove, err := shovey.Get(pathArray[2])
				if err != nil {
					jsonErrorReport(w, r, err.Error(), err.Status())
					return
				}
				sj, err := shove.GetRun(nodeName)
				if err != nil {
					jsonErrorReport(w, r, err.Error(), err.Status())
					return
				}
				err = sj.UpdateFromJSON(sjData)
				if err != nil {
					jsonErrorReport(w, r, err.Error(), err.Status())
					return
				}
				shoveyResponse["id"] = shove.RunID
				shoveyResponse["node"] = nodeName
				shoveyResponse["response"] = "ok" 
			default:
				jsonErrorReport(w, r, "Bad request", http.StatusBadRequest)
				return
			}
		default:
			jsonErrorReport(w, r, "Unrecognized method", http.StatusMethodNotAllowed)
			return
		}
	case "stream":
		if pathArrayLen != 4 {
			jsonErrorReport(w, r, "Bad request", http.StatusBadRequest)
			return
		}
		switch r.Method {
		case "GET":
			var seq int
			r.ParseForm()
			if s, found := r.Form["sequence"]; found {
				if len(s) < 0 {
					jsonErrorReport(w, r, "invalid sequence", http.StatusBadRequest)
					return
				}
				var err error
				seq, err = strconv.Atoi(s[0])
				if err != nil {
					jsonErrorReport(w, r, err.Error(), http.StatusBadRequest)
					return
				}
			}
			var outType string
			if o, found := r.Form["output_type"]; found {
				if len(o) < 0 {
					jsonErrorReport(w, r, "invalid output type", http.StatusBadRequest)
					return
				}
				outType = o[0]
				if outType != "stdout" && outType != "stderr" && outType != "both" {
					jsonErrorReport(w, r, "output type must be 'stdout', 'stderr', or 'both'", http.StatusBadRequest)
					return
				}
			} else {
				outType = "stdout"
			}
			shove, err := shovey.Get(pathArray[2])
			if err != nil {
				jsonErrorReport(w, r, err.Error(), err.Status())
				return
			}
			sj, err := shove.GetRun(pathArray[3])
			if err != nil {
				jsonErrorReport(w, r, err.Error(), err.Status())
				return
			}
			stream, err := sj.GetStreamOutput(outType, seq)
			if err != nil {
				jsonErrorReport(w, r, err.Error(), err.Status())
				return
			}
			combinedOutput, err := sj.CombineStreamOutput(outType, seq)
			if err != nil {
				jsonErrorReport(w, r, err.Error(), err.Status())
				return
			}
			shoveyResponse["run_id"] = sj.ShoveyUUID
			shoveyResponse["node_name"] = sj.NodeName
			shoveyResponse["output_type"] = outType
			if len(stream) != 0 {
				shoveyResponse["last_seq"] = stream[len(stream)-1].Seq
			}
			shoveyResponse["output"] = combinedOutput
		case "PUT":
			streamData, serr := parseObjJSON(r.Body)
			logger.Debugf("streamData: %v", streamData)
			if serr != nil {
				jsonErrorReport(w, r, serr.Error(), http.StatusBadRequest)
				return
			}
			shove, err := shovey.Get(pathArray[2])
			if err != nil {
				jsonErrorReport(w, r, err.Error(), err.Status())
				return
			}
			sj, err := shove.GetRun(pathArray[3])
			if err != nil {
				jsonErrorReport(w, r, err.Error(), err.Status())
				return
			}
			
			output, ok := streamData["output"].(string)
			if !ok {
				oerr := util.Errorf("invalid output")
				jsonErrorReport(w, r, oerr.Error(), oerr.Status())
				return
			}
			outputType, ok := streamData["output_type"].(string)
			if !ok {
				oerr := util.Errorf("invalid output type")
				jsonErrorReport(w, r, oerr.Error(), oerr.Status())
				return
			}

			isLast, ok := streamData["is_last"].(bool)
			if !ok {
				oerr := util.Errorf("invalid is_last")
				jsonErrorReport(w, r, oerr.Error(), oerr.Status())
				return
			}

			seqFloat, ok := streamData["seq"].(float64)
			if !ok {
				oerr := util.Errorf("invalid seq")
				jsonErrorReport(w, r, oerr.Error(), oerr.Status())
				return
			}
			seq := int(seqFloat)

			err = sj.AddStreamOutput(output, outputType, seq, isLast)
			if err != nil {
				jsonErrorReport(w, r, err.Error(), err.Status())
				return
			}
			shoveyResponse["response"] = "ok"
		default:
			jsonErrorReport(w, r, "Unrecognized method", http.StatusMethodNotAllowed)
			return
		}

	default:
		jsonErrorReport(w, r, "Unrecognized operation", http.StatusBadRequest)
		return
	}

	enc := json.NewEncoder(w)
	if jerr := enc.Encode(&shoveyResponse); jerr != nil {
		jsonErrorReport(w, r, jerr.Error(), http.StatusInternalServerError)
	}

	return
}
