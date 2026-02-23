//go:build (!windows && !plan9)

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

// initSyslog will eventually set up using syslog, but for the moment it's a
// no-op until the rest of it is sorted out. Everything about this is subject to
// change.
func initSyslog(useSyslog bool, syslogAddr string) (*log.Logger, error) {
	return nil, nil
}
