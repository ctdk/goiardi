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

package authentication

import (
	"testing"
	"time"
)

func TestTimeSlewSuccess(t *testing.T) {
	dur, _ := time.ParseDuration("15m")
	nowTime := time.Now().UTC()
	nowStr := nowTime.Format(time.RFC3339)
	tok, terr := checkTimeStamp(nowStr, dur)
	if !tok {
		t.Errorf("Time %s should have been OK, but it wasn't!", terr.Error())
	}
	fiveMin := time.Duration(5) * time.Minute
	fiveMinTime := nowTime.Add(fiveMin)
	fStr := fiveMinTime.Format(time.RFC3339)
	tok, terr = checkTimeStamp(fStr, dur)
	if !tok {
		t.Errorf("Time %s five minutes in the future should have been OK, but it wasn't!", terr)
	}
	fiveMinAgoTime := nowTime.Add(-fiveMin)
	fStr = fiveMinAgoTime.Format(time.RFC3339)
	tok, terr = checkTimeStamp(fStr, dur)
	if !tok {
		t.Errorf("Time %s five minutes in the past should have been OK, but it wasn't!", terr)
	}
}

func TestTimeSlewFail(t *testing.T) {
	dur, _ := time.ParseDuration("15m")
	nowTime := time.Now().UTC()
	oneHour := time.Duration(1) * time.Hour
	oneHourTime := nowTime.Add(oneHour)
	oStr := oneHourTime.Format(time.RFC3339)
	tok, terr := checkTimeStamp(oStr, dur)
	if tok {
		t.Errorf("Time %s one hour in the future should have failed, but didn't", terr)
	}
	oneHourAgoTime := nowTime.Add(-oneHour)
	oStr = oneHourAgoTime.Format(time.RFC3339)
	tok, terr = checkTimeStamp(oStr, dur)
	if tok {
		t.Errorf("Time %s one hour in the past should have failed, but didn't", terr)
	}
}
