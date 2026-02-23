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

// Logger tests
package logger

import (
	"fmt"
	"os"
	"path"
	"testing"
)

func TestLoggerInitStdout(t *testing.T) {
	var lName LogLevelName = "debug"
	err := InitializeLogger(lName, "", true, false, "")
	if err != nil {
		t.Errorf("Got an error somehow initializing the logger to stdout: %v", err)
	}
}

func TestLoggerInitFile(t *testing.T) {
	var lName LogLevelName = "debug"
	logDir, err := os.MkdirTemp("", "-logger")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(logDir)

	lf := path.Join(logDir, "init-log")
	err = InitializeLogger(lName, lf, true, false, "")
	if err != nil {
		t.Errorf("Got an error somehow initializing the logger to file %s: %v", lf, err)
	}
}

func TestLoggerInitBadLevel(t *testing.T) {
	var lName LogLevelName = "deborgy"
	err := InitializeLogger(lName, "", true, false, "")
	if err == nil {
		t.Errorf("Initializing logger using bad level %s succeeded when it shouldn't have.", lName)
	}
}

func TestDebug(t *testing.T) {
	var lName LogLevelName = "debug"
	err := InitializeLogger(lName, "", true, false, "")
	if err != nil {
		t.Errorf("Got an error somehow initializing the logger to stdout: %v", err)
	}
	Debugf("foo %d", 1)
	Debug("feeper")
}

func TestInfo(t *testing.T) {
	var lName LogLevelName = "info"
	err := InitializeLogger(lName, "", true, false, "")
	if err != nil {
		t.Errorf("Got an error somehow initializing the logger to stdout: %v", err)
	}
	Debugf("info debug %d", 1)
	Debug("info debug feeper")
	Infof("info info: %d", 6)
	Info("info msg")
}

func TestFatal(t *testing.T) {
	var lName LogLevelName = "fatal"
	err := InitializeLogger(lName, "", true, false, "")
	if err != nil {
		t.Errorf("Got an error somehow initializing the logger to stdout: %v", err)
	}
	Debugf("fatal debug %d", 1)
	Debug("fatal debug feeper")
	Infof("fatal info: %d", 6)
	Info("fatal info msg")

	j := SetFatalExiter(func(){ fmt.Println("would exit here") })
	defer SetFatalExiter(j)

	Fatalf("FATALITY: %d", 666)
	Fatal("FINISH HIM!")
}
