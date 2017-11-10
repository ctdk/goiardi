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

/*
Package loginfo tracks changes to objects when they're saved, noting the actor performing the action, what kind of action it was, the time of the change, the type of object and its id, and a dump of the object's state. */
package loginfo

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ctdk/goiardi/actor"
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/datastore"
	"github.com/ctdk/goiardi/serfin"
	"github.com/ctdk/goiardi/util"
	"github.com/tideland/golib/logger"
	diff "github.com/yudai/gojsondiff"
	"github.com/yudai/gojsondiff/formatter"
)

var noEventFound = errors.New("event not found")

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
	Diff string `json:"diff"`
	ID           int         `json:"id"`
}

// LogEvent writes an event of the action type, performed by the given actor,
// against the given object.
func LogEvent(doer actor.Actor, obj util.GoiardiObj, action string) error {
	if !config.Config.LogEvents {
		logger.Debugf("Not logging this event")
		return nil
	}
	logger.Debugf("Logging event")

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
	extInfo, err := datastore.EncodeToJSON(obj)
	if err != nil {
		return err
	}

	// get the most recent logged event, if one is present
	mostRecent, err := getMostRecentEvent(le.ObjectName, le.ObjectType)

	// If there is a logged event for this object in the database:
	// make a diff of the json between the object at the last event and
	// this one, clear the extended info for the previous event, and save.
	if err != nil && err != noEventFound {
		return err
	} else if err == nil { // don't try making a diff if there's no previous
		oldExtInfo := mostRecent.ExtendedInfo
		differ := diff.New()
		df, err := differ.Compare([]byte(oldExtInfo), []byte(extInfo))
		if err != nil {
			return err
		}

		fmtr := formatter.NewDeltaFormatter()
		fmtr.PrintIndent = false

		dstr, err := fmtr.Format(df)
		// shouldn't be able to happen, but...
		if err != nil {
			return err
		}

		le.Diff = dstr
	}

	le.ExtendedInfo = extInfo
	actorInfo, err := datastore.EncodeToJSON(doer)
	if err != nil {
		return err
	}
	le.ActorInfo = actorInfo
	if config.Config.SerfEventAnnounce {
		qle := make(map[string]interface{}, 4)
		qle["time"] = le.Time
		qle["action"] = le.Action
		qle["object_type"] = le.ObjectType
		qle["object_name"] = le.ObjectName
		go serfin.SendEvent("log-event", qle)
	}

	if mostRecent != nil {
		err = mostRecent.clearExtendedInfo()
		if err != nil {
			return err
		}
	}

	if config.UsingDB() {
		return le.writeEventSQL()
	}
	return le.writeEventInMem()
}

// Import a log info event from an export dump.
func Import(logData map[string]interface{}) error {
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

	if config.UsingDB() {
		return le.importEventSQL()
	}
	return le.importEventInMem()
}

func (le *LogInfo) writeEventInMem() error {
	ds := datastore.New()
	return ds.SetLogInfo(le)
}

func (le *LogInfo) importEventInMem() error {
	ds := datastore.New()
	return ds.SetLogInfo(le, le.ID)
}

func (le *LogInfo) clearExtendedInfo() error {
	if config.UsingDB() {
		return le.clearExtendedInfoSQL()
	}

	// clear this in in-mem too, eh. At least we can use an existing
	// method, although it may be better to rename it.
	le.ExtendedInfo = ""
	return le.importEventInMem()

	return nil 
}

// Get a particular event by its id.
func Get(id int) (*LogInfo, error) {
	var le *LogInfo

	if config.UsingDB() {
		var err error
		le, err = getLogEventSQL(id)
		if err != nil {
			if err == sql.ErrNoRows {
				err = fmt.Errorf("Couldn't find log event with id %d", id)
			}
			return nil, err
		}
	} else {
		ds := datastore.New()
		c, err := ds.GetLogInfo(id)
		if err != nil {
			return nil, err
		}
		if c != nil {
			le = c.(*LogInfo)
			le.ID = id
		}
	}
	return le, nil
}

// DoesExist checks if the particular event in question exists. To be compatible
// with the interface for HEAD responses, this method receives a string rather
// than an integer.
func DoesExist(eventID string) (bool, util.Gerror) {
	id, err := strconv.Atoi(eventID)
	if err != nil {
		cerr := util.CastErr(err)
		return false, cerr
	}
	if config.UsingDB() {
		found, err := checkLogEventSQL(id)
		if err != nil {
			cerr := util.CastErr(err)
			return false, cerr
		}
		return found, nil
	}

	ds := datastore.New()
	c, err := ds.GetLogInfo(id)
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
	ds.DeleteLogInfo(le.ID)
	return nil
}

// PurgeLogInfos removes all logged events before the given id.
func PurgeLogInfos(id int) (int64, error) {
	if config.UsingDB() {
		return purgeSQL(id)
	}
	ds := datastore.New()
	return ds.PurgeLogInfoBefore(id)
}

// GetLogInfos gets a slice of the logged events. May be called with an offset
// and limit, (in that order) but that is not required. The offset can be
// specified without a limit, but a limit requires an offset (which can be 0).
// The map of search params may be nil, but something must be present.
func GetLogInfos(searchParams map[string]string, limits ...int) ([]*LogInfo, error) {
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
		return getLogInfoListSQL(searchParams, from, until, limits...)
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
	arr := ds.GetLogInfoList()
	lis := make([]*LogInfo, len(arr))
	var keys []int
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

func getMostRecentEvent(name string, objectType string) (*LogInfo, error) {
	if config.UsingDB() {
		e, err := getMostRecentEventSQL(name, objectType)
		if err != nil && err == sql.ErrNoRows {
			err = noEventFound
		}
		return e, err
	}
	// It remains to be seen if this will be too terrible for words for the
	// in-mem log info storage, or if the performance will end up being
	// acceptable. A separate index may end up being necessary.
	ds := datastore.New()
	arr := ds.GetLogInfoList()
	var keys []int
	for k := range arr {
		keys = append(keys, k)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(keys)))
	var mRecent *LogInfo
	for _, i := range keys {
		if k, ok := arr[i]; ok {
			item := k.(*LogInfo)
			if item.ObjectType == objectType && item.ObjectName == name {
				item.ID = i
				mRecent = item
				break
			}
		}
	}
	if mRecent == nil {
		return nil, noEventFound
	}
	return mRecent, nil
}

func (le *LogInfo) checkTimeRange(from, until time.Time) bool {
	return le.Time.After(from) && le.Time.Before(until)
}

// AllLogInfos returns a list of all logged events in the database. Provides a
// wrapper around GetLogInfos() for consistency with the other object types for
// exporting data.
func AllLogInfos() []*LogInfo {
	l, _ := GetLogInfos(nil)
	return l
}
