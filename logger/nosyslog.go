//go:build windows || plan9

/*
 * Copyright (c) 2013-2026, Jeremy Bingham (<jeremy@goiardi.gl>)
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

package logger

import (
	"log"
)

// initSyslog, for platforms that do not use syslog at all, will always be a
// no-op. Even so, this is even more of a dummy function than it would be until
// syslog is sorted out where it's available.
func initSyslog(useSyslog bool, syslogAddr string) (*log.Logger, error) {
	return nil, nil
}
