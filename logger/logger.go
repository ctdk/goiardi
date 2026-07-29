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

/*
Package logger implements a structured logger for goiardi. The previous and excellent former logger from github.com/tideland/golib/logger that goiardi used has unfortunately lain fallow for a long time and causes some problems with a new check that golang runs with 'go vet' and during 'go test' that throws errors when non-constant strings are used with formatted error functions. The tideland logger does not provide non-formatted error functions along the lines of 'logger.Error', so this package was created.
*/
package logger

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

type LogLevel slog.Level

const (
	LevelDebug    LogLevel = LogLevel(slog.LevelDebug)
	LevelInfo              = LogLevel(slog.LevelInfo)
	LevelNotice            = LogLevel(slog.Level(2))
	LevelWarning           = LogLevel(slog.LevelWarn)
	LevelError             = LogLevel(slog.LevelError)
	LevelCritical          = LogLevel(slog.Level(10))
	LevelFatal             = LogLevel(slog.Level(12))
)

// CurrentLogLevel is the current log level. Useful if you need to conditionally
// execute expensive operations to log.
var CurrentLogLevel LogLevel

// Taking a hint from the tideland golib logger library here for fatal exits.

// FatalExiterFunc is a type to define a function that runs after a call to
// Fatalf or Fatal. Normally it just exits with error code -1, but it can be
// reset to a different function for testing and the like

type FatalExiterFunc func()

// OsFatalExiter exits the application with error code -1.
func OsFatalExiter() {
	os.Exit(-1)
}

var (
	logMux         sync.RWMutex
	logFatalExiter = OsFatalExiter
)

// LogLevelName gives convenient, easier to remember than number name for the
// different levels of logging. Subject to tweaking.
type LogLevelName string

const (
	DebugLevel    LogLevelName = "DEBUG"
	InfoLevel                  = "INFO"
	NoticeLevel                = "NOTICE"
	WarningLevel               = "WARNING"
	ErrorLevel                 = "ERROR"
	CriticalLevel              = "CRITICAL"
	FatalLevel                 = "FATAL"
)

// DefaultLevel, shockingly, is the default logging level if nothing's been
// specified.
const DefaultLevel = FatalLevel

var (
	slogDebug    = slog.StringValue(string(DebugLevel))
	slogInfo     = slog.StringValue(string(InfoLevel))
	slogNotice   = slog.StringValue(string(NoticeLevel))
	slogWarning  = slog.StringValue(string(WarningLevel))
	slogError    = slog.StringValue(string(ErrorLevel))
	slogCritical = slog.StringValue(string(CriticalLevel))
	slogFatal    = slog.StringValue(string(FatalLevel))
)

// LogLevelNames maps the name of a log level to its numerical representation.
var LogLevelNames = map[LogLevelName]LogLevel{DebugLevel: LevelDebug, InfoLevel: LevelInfo, NoticeLevel: LevelNotice, WarningLevel: LevelWarning, ErrorLevel: LevelError, CriticalLevel: LevelCritical, FatalLevel: LevelFatal}

// LogModePerm is the permissions mask for the log file. Currently owner
// read+write, group and world read.
const LogModePerm = 0644

// InitializeLogger configures the structured logger. Set 'useJsonOutput' to
// true to get JSON formatted log statements.
func InitializeLogger(logLevel LogLevelName, logFile string, useJsonOutput bool, useSyslog bool, syslogAddr string) error {
	logLevel = LogLevelName(strings.ToUpper(string(logLevel)))
	if logLevel == "" {
		logLevel = DefaultLevel
	}

	// will probably need to deal with syslog around here when the time
	// comes.
	var logFp *os.File
	if logFile != "" {
		var err error
		logFp, err = os.OpenFile(logFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, os.ModeAppend|LogModePerm)
		if err != nil {
			return err
		}
	} else {
		// just log to stdout
		logFp = os.Stdout
	}

	lvl, ok := LogLevelNames[logLevel]
	if !ok {
		return fmt.Errorf("Log level '%s' is not a valid log level.", logLevel)
	}

	CurrentLogLevel = lvl

	// handler options
	handlerOpts := &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.Level(lvl),
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.LevelKey {
				lev := LogLevel(a.Value.Any().(slog.Level))
				switch {
				case lev < LevelInfo:
					a.Value = slogDebug
				case lev < LevelNotice:
					a.Value = slogInfo
				case lev < LevelWarning:
					a.Value = slogNotice
				case lev < LevelError:
					a.Value = slogWarning
				case lev < LevelCritical:
					a.Value = slogError
				case lev < LevelFatal:
					a.Value = slogCritical
				default:
					a.Value = slogFatal
				}
			}
			return a
		},
	}

	var h slog.Handler
	if useJsonOutput {
		h = slog.NewJSONHandler(logFp, handlerOpts)
	} else {
		h = slog.NewTextHandler(logFp, handlerOpts)
	}

	l := slog.New(h)
	slog.SetDefault(l)

	// and this should all be done.
	return nil
}

// VerboseFlagToLevel converts the number of -V arguments to the appropriate
// LogLevelName.
func VerboseFlagToLevel(vnum int) LogLevelName {
	// If the vnum is higher than what would give DebugLevel, set it to
	// that. It's len(LogLevelNames) - 1 because Fatal is no -Vs at all.
	if vnum > len(LogLevelNames)-1 {
		vnum = len(LogLevelNames) - 1
	}

	type kv struct {
		Key LogLevelName
		Val LogLevel
	}
	ss := make([]kv, len(LogLevelNames))
	u := 0
	for k, v := range LogLevelNames {
		ss[u] = kv{k, v}
		u++
	}
	sort.Slice(ss, func(i, j int) bool {
		return ss[i].Val > ss[j].Val
	})

	lname := ss[vnum]
	return lname.Key
}

// Base log statement function used by these convenient helper functions.
func logStatement(msg string, lvl LogLevel, skip int) {
	l := slog.Default()
	var pcs [1]uintptr
	runtime.Callers(3+skip, pcs[:]) // skip Callers, the calling wrapper, & this
	r := slog.NewRecord(time.Now(), slog.Level(lvl), msg, pcs[0])
	_ = l.Handler().Handle(context.Background(), r)
}

// LogSkip logs a message at the given level with an additional number of
// calling functions to skip for the source code line in the log statement.
func LogSkip(msg string, lvl LogLevel, skip int) {
	logStatement(msg, lvl, skip)
}

// Debugf logs a formatted message at the debug level.
func Debugf(format string, args ...interface{}) {
	l := slog.Default()
	if !l.Enabled(context.Background(), slog.Level(LevelDebug)) {
		return
	}
	msg := fmt.Sprintf(format, args...)
	logStatement(msg, LevelDebug, 0)
}

// Debug logs an unformatted message at the debug level.
func Debug(msg string) {
	l := slog.Default()
	if !l.Enabled(context.Background(), slog.Level(LevelDebug)) {
		return
	}
	logStatement(msg, LevelDebug, 0)
}

// Infof logs a formatted message at the debug level.
func Infof(format string, args ...interface{}) {
	l := slog.Default()
	if !l.Enabled(context.Background(), slog.Level(LevelInfo)) {
		return
	}
	msg := fmt.Sprintf(format, args...)
	logStatement(msg, LevelInfo, 0)
}

// Info logs an unformatted message at the debug level.
func Info(msg string) {
	l := slog.Default()
	if !l.Enabled(context.Background(), slog.Level(LevelInfo)) {
		return
	}
	logStatement(msg, LevelInfo, 0)
}

// Noticef logs a formatted message at the debug level.
func Noticef(format string, args ...interface{}) {
	l := slog.Default()
	if !l.Enabled(context.Background(), slog.Level(LevelNotice)) {
		return
	}
	msg := fmt.Sprintf(format, args...)
	logStatement(msg, LevelNotice, 0)
}

// Notice logs an unformatted message at the debug level.
func Notice(msg string) {
	l := slog.Default()
	if !l.Enabled(context.Background(), slog.Level(LevelNotice)) {
		return
	}
	logStatement(msg, LevelNotice, 0)
}

// Warningf logs a formatted message at the debug level.
func Warningf(format string, args ...interface{}) {
	l := slog.Default()
	if !l.Enabled(context.Background(), slog.Level(LevelWarning)) {
		return
	}
	msg := fmt.Sprintf(format, args...)
	logStatement(msg, LevelWarning, 0)
}

// Warning logs an unformatted message at the debug level.
func Warning(msg string) {
	l := slog.Default()
	if !l.Enabled(context.Background(), slog.Level(LevelWarning)) {
		return
	}
	logStatement(msg, LevelWarning, 0)
}

// Errorf logs a formatted message at the debug level.
func Errorf(format string, args ...interface{}) {
	l := slog.Default()
	if !l.Enabled(context.Background(), slog.Level(LevelError)) {
		return
	}
	msg := fmt.Sprintf(format, args...)
	logStatement(msg, LevelError, 0)
}

// Error logs an unformatted message at the debug level.
func Error(msg string) {
	l := slog.Default()
	if !l.Enabled(context.Background(), slog.Level(LevelError)) {
		return
	}
	logStatement(msg, LevelError, 0)
}

// Criticalf logs a formatted message at the debug level.
func Criticalf(format string, args ...interface{}) {
	l := slog.Default()
	if !l.Enabled(context.Background(), slog.Level(LevelCritical)) {
		return
	}
	msg := fmt.Sprintf(format, args...)
	logStatement(msg, LevelCritical, 0)
}

// Critical logs an unformatted message at the debug level.
func Critical(msg string) {
	l := slog.Default()
	if !l.Enabled(context.Background(), slog.Level(LevelCritical)) {
		return
	}
	logStatement(msg, LevelCritical, 0)
}

// Fatalf logs a formatted message at the debug level.
func Fatalf(format string, args ...interface{}) {
	l := slog.Default()
	if !l.Enabled(context.Background(), slog.Level(LevelFatal)) {
		return
	}
	msg := fmt.Sprintf(format, args...)
	logStatement(msg, LevelFatal, 0)
	logFatalExiter()
}

// Fatal logs an unformatted message at the debug level.
func Fatal(msg string) {
	l := slog.Default()
	if !l.Enabled(context.Background(), slog.Level(LevelFatal)) {
		return
	}
	logStatement(msg, LevelFatal, 0)
	logFatalExiter()
}

// SetFatalExiter sets the final exiter function and returns the current one.
// Adapted from the tideland golib logger, of course.
func SetFatalExiter(f FatalExiterFunc) FatalExiterFunc {
	logMux.Lock()
	defer logMux.Unlock()
	c := logFatalExiter
	logFatalExiter = f
	return c
}

// SetLevel sets the logging level and returns the former log level.
func SetLevel(level LogLevelName) LogLevelName {
	lvl, _ := LogLevelNames[level]
	oldLevel := LogLevel(slog.SetLogLoggerLevel(slog.Level(lvl)))
	CurrentLogLevel = lvl
	// reverse LogLevelNames
	l := make(map[LogLevel]LogLevelName)
	for k, v := range LogLevelNames {
		l[v] = k
	}

	return l[oldLevel]
}
