/*
 * Copyright (c) 2013-2019, Jeremy Bingham (<jeremy@goiardi.gl>)
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
Package loginfo tracks changes to objects when they're saved, noting the actor performing the action, what kind of action it was, the time of the change, the type of object and its id, and a dump of the object's state. */
package loginfo

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ctdk/goiardi/actor"
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/datastore"
	"github.com/ctdk/goiardi/organization"
	"github.com/ctdk/goiardi/serfin"
	"github.com/ctdk/goiardi/util"
	"github.com/tideland/golib/logger"
)

// LogInfo holds log information about events.
type LogInfo struct {
	Actor        actor.Actor `json:"-"`
	ActorInfo    string      `json:"actor_info"`
	ActorType    string      `json:"actor_type"`
	Time         time.Time   `json:"time"`
	Action       string      `json:"action"`
	ObjectType   string      `json:"object_type"`
	ObjectName   string      `json:"object_name"`
	ExtendedInfo string      `json:"extended_info"`
	ID           int         `json:"id"`
	org          *organization.Organization
}

// TODO: make me a config option
const debugLogEvent = false

// LogEvent writes an event of the action type, performed by the given actor,
// against the given object.
func LogEvent(org *organization.Organization, doer actor.Actor, obj util.GoiardiObj, action string) error {
	if !config.Config.LogEvents {
		if debugLogEvent {
			logger.Debugf("Not logging this event")
		}
		return nil
	}

	if debugLogEvent {
		logger.Debugf("Logging event")
	}

	var actorType string
	if doer.IsUser() {
		actorType = "user"
	} else {
		actorType = "client"
	}
	le := new(LogInfo)
	le.Action = action
	le.Actor = doer
	le.ActorType = actorType
	le.ObjectName = obj.GetName()
	le.ObjectType = reflect.TypeOf(obj).String()
	le.Time = time.Now()

	if !config.Config.SkipLogExtended {
		extInfo, err := datastore.EncodeToJSON(obj)
		if err != nil {
			return err
		}
		le.ExtendedInfo = extInfo
	}

	actorInfo, err := datastore.EncodeToJSON(doer)
	if err != nil {
		return err
	}
	le.ActorInfo = actorInfo
	le.org = org
	var orgName string
	if org != nil {
		orgName = org.Name
	}

	if config.Config.SerfEventAnnounce {
		qle := make(map[string]interface{}, 4)
		qle["time"] = le.Time
		qle["action"] = le.Action
		qle["object_type"] = le.ObjectType
		qle["object_name"] = le.ObjectName
		qle["organization"] = orgName
		go serfin.SendEvent("log-event", qle)
	}

	if config.UsingDB() {
		return le.writeEventSQL()
	}
	return le.writeEventInMem()
}

// Import a log info event from an export dump.
func Import(org *organization.Organization, logData map[string]interface{}) error {
	le := new(LogInfo)
	le.Action = logData["action"].(string)
	le.ActorType = logData["actor_type"].(string)
	le.ActorInfo = logData["actor_info"].(string)
	le.ObjectType = logData["object_type"].(string)
	le.ObjectName = logData["object_name"].(string)
	le.ExtendedInfo = logData["extended_info"].(string)
	le.ID = int(logData["id"].(float64))
	switch l := logData["id"].(type) {
	case float64:
		le.ID = int(l)
	case json.Number:
		k, _ := l.Int64()
		le.ID = int(k)
	}
	t, err := time.Parse(time.RFC3339, logData["time"].(string))
	if err != nil {
		return nil
	}
	le.Time = t
	le.org = org

	if config.UsingDB() {
		return le.importEventSQL()
	}
	return le.importEventInMem()
}

func (le *LogInfo) writeEventInMem() error {
	ds := datastore.New()
	return ds.SetLogInfo(le.org.Name, le)
}

func (le *LogInfo) importEventInMem() error {
	ds := datastore.New()
	return ds.SetLogInfo(le.org.Name, le, le.ID)
}

// Get a particular event by its id.
func Get(org *organization.Organization, id int) (*LogInfo, error) {
	var le *LogInfo

	orgId, orgName := getOrgInfo(org)

	if config.UsingDB() {
		var err error
		le, err = getLogEventSQL(id, orgId)
		if err != nil {
			if err == sql.ErrNoRows {
				err = fmt.Errorf("Couldn't find log event with id %d", id)
			}
			return nil, err
		}
	} else {
		ds := datastore.New()
		c, err := ds.GetLogInfo(orgName, id)
		if err != nil {
			return nil, err
		}
		if c != nil {
			le = c.(*LogInfo)
			le.ID = id
			le.org = org
		}
	}
	return le, nil
}

// DoesExist checks if the particular event in question exists. To be compatible
// with the interface for HEAD responses, this method receives a string rather
// than an integer.
func DoesExist(org *organization.Organization, eventID string) (bool, util.Gerror) {
	id, err := strconv.Atoi(eventID)
	if err != nil {
		cerr := util.CastErr(err)
		return false, cerr
	}

	orgId, orgName := getOrgInfo(org)

	if config.UsingDB() {
		found, err := checkLogEventSQL(id, orgId)
		if err != nil {
			cerr := util.CastErr(err)
			return false, cerr
		}
		return found, nil
	}

	ds := datastore.New()
	c, err := ds.GetLogInfo(orgName, id)
	if err != nil {
		cerr := util.CastErr(err)
		return false, cerr
	}
	var found bool
	if c != nil {
		found = true
	}
	return found, nil
}

// Delete a logged event.
func (le *LogInfo) Delete() error {
	if config.UsingDB() {
		return le.deleteSQL()
	}
	ds := datastore.New()
	ds.DeleteLogInfo(le.OrgName(), le.ID)
	return nil
}

// PurgeLogInfos removes all logged events before the given id.
func PurgeLogInfos(org *organization.Organization, id int) (int64, error) {
	orgId, orgName := getOrgInfo(org)
	if config.UsingDB() {
		return purgeSQL(id, orgId)
	}
	ds := datastore.New()
	return ds.PurgeLogInfoBefore(orgName, id)
}

// GetLogInfos gets a slice of the logged events. May be called with an offset
// and limit, (in that order) but that is not required. The offset can be
// specified without a limit, but a limit requires an offset (which can be 0).
// The map of search params may be nil, but something must be present.
func GetLogInfos(org *organization.Organization, searchParams map[string]string, limits ...int) ([]*LogInfo, error) {
	orgId, orgName := getOrgInfo(org)
	// optional params
	var from, until time.Time
	if f, ok := searchParams["from"]; ok {
		fUnix, err := strconv.ParseInt(f, 10, 64)
		if err != nil {
			return nil, err
		}
		from = time.Unix(fUnix, 0)
	} else {
		from = time.Unix(0, 0)
	}
	if u, ok := searchParams["until"]; ok {
		uUnix, err := strconv.ParseInt(u, 10, 64)
		if err != nil {
			return nil, err
		}
		until = time.Unix(uUnix, 0)
	} else {
		until = time.Now()
	}
	if ot, ok := searchParams["object_type"]; ok {
		/* If this is false, assume it's not a name of the pointer */
		if !strings.ContainsAny(ot, "*.") {
			if ot == "environment" {
				searchParams["object_type"] = "*environment.ChefEnvironment"
			} else if ot == "cookbook_version" {
				searchParams["object_type"] = "*cookbook.CookbookVersion"
			} else {
				searchParams["object_type"] = fmt.Sprintf("*%s.%s", ot, strings.Title(ot))
			}
		}
	}
	if config.UsingDB() {
		return getLogInfoListSQL(orgId, searchParams, from, until, limits...)
	}
	var offset, limit int
	if len(limits) > 0 {
		offset = limits[0]
		if len(limits) > 1 {
			limit = limits[1]
		}
	} else {
		offset = 0
	}

	ds := datastore.New()
	arr := ds.GetLogInfoList(orgName)
	lis := make([]*LogInfo, len(arr))
	keys := make([]int, 0, len(arr))
	for k := range arr {
		keys = append(keys, k)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(keys)))
	n := 0
	for _, i := range keys {
		k, ok := arr[i]
		if ok {
			item := k.(*LogInfo)
			if item.checkTimeRange(from, until) && (searchParams["action"] == "" || searchParams["action"] == item.Action) && (searchParams["object_type"] == "" || searchParams["object_type"] == item.ObjectType) && (searchParams["object_name"] == "" || searchParams["object_name"] == item.ObjectName) && (searchParams["doer"] == "" || searchParams["doer"] == item.Actor.GetName()) {
				item.ID = i
				lis[n] = item
				n++
			}
		}
	}
	if len(lis) == 0 {
		return lis, nil
	}
	if len(limits) > 1 {
		limit = offset + limit
		if limit > len(lis) {
			limit = len(lis)
		}
	} else {
		limit = len(lis)
	}
	if n < limit {
		limit = n
	}
	return lis[offset:limit], nil
}

func (le *LogInfo) checkTimeRange(from, until time.Time) bool {
	return le.Time.After(from) && le.Time.Before(until)
}

// AllLogInfos returns a list of all logged events in the database. Provides a
// wrapper around GetLogInfos() for consistency with the other object types for
// exporting data.
func AllLogInfos(org *organization.Organization) []*LogInfo {
	l, _ := GetLogInfos(org, nil)
	return l
}

func getOrgName(org *organization.Organization) string {
	if org != nil {
		return org.Name
	}
	return ""
}

func (le *LogInfo) GetName() string {
	return strconv.Itoa(le.ID)
}

func (le *LogInfo) ContainerType() string {
	return "log-infos"
}

func (le *LogInfo) ContainerKind() string {
	return "containers"
}

func (le *LogInfo) OrgName() string {
	if le.org != nil {
		return le.org.Name
	}
	return ""
}

func (le *LogInfo) OrgId() int64 {
	if le.org != nil {
		return le.org.GetId()
	}
	return 0
}

// This is a helper function to get an org's id and name if the org passed in
// is not nil.
func getOrgInfo(org *organization.Organization) (int64, string) {
	var orgName string
	var orgId int64

	if org != nil {
		if config.UsingDB() {
			orgId = org.GetId()
		}
		orgName = org.Name
	}
	return orgId, orgName
}
