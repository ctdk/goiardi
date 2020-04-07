// Tideland Go Library - Logger
//
// Copyright (C) 2012-2017 Frank Mueller / Tideland / Oldenburg / Germany
//
// All rights reserved. Use of this source code is governed
// by the new BSD license.

package logger

//--------------------
// IMPORTS
//--------------------

import (
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"runtime"
	"strings"
	"sync"
	"time"
)

//--------------------
// TYPES AND TYPE FUNCTIONS
//--------------------

// LogLevel describes the chosen log level between
// debug and critical.
type LogLevel int

// Log levels to control the logging output.
const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarning
	LevelError
	LevelCritical
	LevelFatal
)

// FatalExiterFunc defines a functions that will be called
// in case of a Fatalf call.
type FatalExiterFunc func()

// OsFatalExiter exits the application with os.Exit and
// the return code -1.
func OsFatalExiter() {
	os.Exit(-1)
}

// PanicFatalExiter exits the application with a panic.
func PanicFatalExiter() {
	panic("program aborted after fatal situation, see log")
}

// FilterFunc allows to filter the output of the logging. Filters
// have to return true if the received entry shall be filtered and
// not output.
type FilterFunc func(level LogLevel, info, msg string) bool

//--------------------
// LOG CONTROL
//--------------------

// Log control variables.
var (
	logMux         sync.RWMutex
	logLevel       LogLevel        = LevelInfo
	logBackend     Logger          = NewStandardLogger(os.Stdout)
	logFatalExiter FatalExiterFunc = OsFatalExiter
	logFilter      FilterFunc
)

// Level returns the current log level.
func Level() LogLevel {
	logMux.Lock()
	defer logMux.Unlock()
	return logLevel
}

// SetLevel switches to a new log level and returns
// the current one.
func SetLevel(level LogLevel) LogLevel {
	logMux.Lock()
	defer logMux.Unlock()
	current := logLevel
	switch {
	case level <= LevelDebug:
		logLevel = LevelDebug
	case level >= LevelFatal:
		logLevel = LevelFatal
	default:
		logLevel = level
	}
	return current
}

// SetLevelString switches to a new log level passed as
// string. Accepted are the values "debug", "info", "warning"
// "error", "critical", and "fatal". The passed string will
// be set to lower-case. The function is intended to be used
// when then log level is read out of a configuration.
func SetLevelString(levelstr string) LogLevel {
	logMux.Lock()
	defer logMux.Unlock()
	current := logLevel
	switch strings.ToLower(levelstr) {
	case "debug":
		logLevel = LevelDebug
	case "info":
		logLevel = LevelInfo
	case "warning":
		logLevel = LevelWarning
	case "error":
		logLevel = LevelError
	case "critical":
		logLevel = LevelCritical
	case "fatal":
		logLevel = LevelFatal
	}
	return current
}

// SetLogger sets a new logger.
func SetLogger(l Logger) Logger {
	logMux.Lock()
	defer logMux.Unlock()
	old := logBackend
	logBackend = l
	return old
}

// SetFatalExiter sets the fatal exiter function and
// returns the current one.
func SetFatalExiter(fef FatalExiterFunc) FatalExiterFunc {
	logMux.Lock()
	defer logMux.Unlock()
	current := logFatalExiter
	logFatalExiter = fef
	return current
}

// SetFilter sets the global output filter and returns
// the current one.
func SetFilter(ff FilterFunc) FilterFunc {
	logMux.Lock()
	defer logMux.Unlock()
	current := logFilter
	logFilter = ff
	return current
}

// UnsetFilter removes the global output folter and
// returns the current one.
func UnsetFilter() FilterFunc {
	logMux.Lock()
	defer logMux.Unlock()
	current := logFilter
	logFilter = nil
	return current
}

//--------------------
// LOGGING
//--------------------

// Debugf logs a message at debug level.
func Debugf(format string, args ...interface{}) {
	logMux.RLock()
	defer logMux.RUnlock()
	if logLevel <= LevelDebug {
		info := retrieveCallInfo().verboseFormat()
		msg := fmt.Sprintf(format, args...)

		if shallLog(LevelDebug, info, msg) {
			logBackend.Debug(info, msg)
		}
	}
}

// Infof logs a message at info level.
func Infof(format string, args ...interface{}) {
	logMux.RLock()
	defer logMux.RUnlock()
	if logLevel <= LevelInfo {
		info := retrieveCallInfo().shortFormat()
		msg := fmt.Sprintf(format, args...)

		if shallLog(LevelInfo, info, msg) {
			logBackend.Info(info, msg)
		}
	}
}

// Warningf logs a message at warning level.
func Warningf(format string, args ...interface{}) {
	logMux.RLock()
	defer logMux.RUnlock()
	if logLevel <= LevelWarning {
		info := retrieveCallInfo().shortFormat()
		msg := fmt.Sprintf(format, args...)

		if shallLog(LevelWarning, info, msg) {
			logBackend.Warning(info, msg)
		}
	}
}

// Errorf logs a message at error level.
func Errorf(format string, args ...interface{}) {
	logMux.RLock()
	defer logMux.RUnlock()
	if logLevel <= LevelError {
		info := retrieveCallInfo().shortFormat()
		msg := fmt.Sprintf(format, args...)

		if shallLog(LevelError, info, msg) {
			logBackend.Error(info, msg)
		}
	}
}

// Criticalf logs a message at critical level.
func Criticalf(format string, args ...interface{}) {
	logMux.RLock()
	defer logMux.RUnlock()
	if logLevel <= LevelCritical {
		info := retrieveCallInfo().verboseFormat()
		msg := fmt.Sprintf(format, args...)

		if shallLog(LevelCritical, info, msg) {
			logBackend.Critical(info, msg)
		}
	}
}

// Fatalf logs a message independent of any level. After
// logging the message the functions calls the fatal exiter
// function, which by default means exiting the application
// with error code -1.
func Fatalf(format string, args ...interface{}) {
	logMux.RLock()
	defer logMux.RUnlock()
	info := retrieveCallInfo().verboseFormat()
	msg := fmt.Sprintf(format, args...)

	logBackend.Fatal(info, msg)
	logFatalExiter()
}

//--------------------
// LOGGER INTERFACE
//--------------------

// Logger is the interface for different logger backends.
type Logger interface {
	// Debug logs a debugging message.
	Debug(info, msg string)

	// Info logs an informational message.
	Info(info, msg string)

	// Warning logs a warning message.
	Warning(info, msg string)

	// Error logs an error message.
	Error(info, msg string)

	// Critical logs a critical error message.
	Critical(info, msg string)

	// Fatal logs a fatal error message.
	Fatal(info, msg string)
}

// defaultTimeFormat controls how the timestamp of the standard
// logger is printed by default.
const defaultTimeFormat = "2006-01-02 15:04:05 Z07:00"

//--------------------
// STANDARD LOGGER
//--------------------

// standardLogger is a simple logger writing to the given writer. Beside
// the output it doesn't handle the levels differently.
type standardLogger struct {
	mutex      sync.Mutex
	out        io.Writer
	timeFormat string
}

// NewTimeformatLogger creates a logger writing to the passed
// output and with the time formatted with the passed time format.
func NewTimeformatLogger(out io.Writer, timeFormat string) Logger {
	return &standardLogger{
		out:        out,
		timeFormat: timeFormat,
	}
}

// NewStandardLogger creates the standard logger writing
// to the passed output.
func NewStandardLogger(out io.Writer) Logger {
	return NewTimeformatLogger(out, defaultTimeFormat)
}

// Debug implements Logger.
func (sl *standardLogger) Debug(info, msg string) {
	sl.writeLog("DEBUG", info, msg)
}

// Info implements Logger.
func (sl *standardLogger) Info(info, msg string) {
	sl.writeLog("INFO", info, msg)
}

// Warning implements Logger.
func (sl *standardLogger) Warning(info, msg string) {
	sl.writeLog("WARNING", info, msg)
}

// Error implements Logger.
func (sl *standardLogger) Error(info, msg string) {
	sl.writeLog("ERROR", info, msg)
}

// Critical implements Logger.
func (sl *standardLogger) Critical(info, msg string) {
	sl.writeLog("CRITICAL", info, msg)
}

// Fatal implements Logger.
func (sl *standardLogger) Fatal(info, msg string) {
	sl.writeLog("FATAL", info, msg)
}

// writeLog writes the log output.
func (sl *standardLogger) writeLog(level, info, msg string) {
	sl.mutex.Lock()
	defer sl.mutex.Unlock()

	io.WriteString(sl.out, time.Now().Format(sl.timeFormat))
	io.WriteString(sl.out, " [")
	io.WriteString(sl.out, level)
	io.WriteString(sl.out, "] ")
	io.WriteString(sl.out, info)
	io.WriteString(sl.out, " ")
	io.WriteString(sl.out, msg)
	io.WriteString(sl.out, "\n")
}

// goLogger just uses the standard go log package.
type goLogger struct{}

// NewGoLogger returns a logger implementation using the
// Go log package.
func NewGoLogger() Logger {
	return &goLogger{}
}

// Debug is specified on the Logger interface.
func (gl *goLogger) Debug(info, msg string) {
	log.Println("[DEBUG]", info, msg)
}

// Info is specified on the Logger interface.
func (gl *goLogger) Info(info, msg string) {
	log.Println("[INFO]", info, msg)
}

// Warning is specified on the Logger interface.
func (gl *goLogger) Warning(info, msg string) {
	log.Println("[WARNING]", info, msg)
}

// Error is specified on the Logger interface.
func (gl *goLogger) Error(info, msg string) {
	log.Println("[ERROR]", info, msg)
}

// Critical is specified on the Logger interface.
func (gl *goLogger) Critical(info, msg string) {
	log.Println("[CRITICAL]", info, msg)
}

// Fatal is specified on the Logger interface.
func (gl *goLogger) Fatal(info, msg string) {
	log.Println("[FATAL]", info, msg)
}

// TestLogger extends the Logger interface with methods to retrieve
// and reset the collected data.
type TestLogger interface {
	Logger

	// Len returns the number of collected entries.
	Len() int

	// Entries returns the collected entries.
	Entries() []string

	// Reset clears the collected entries.
	Reset()
}

// testLogger simply collects logs to be evaluated inside of tests.
type testLogger struct {
	mux     sync.Mutex
	entries []string
}

// NewTestLogger returns a special logger for testing.
func NewTestLogger() TestLogger {
	return &testLogger{}
}

// Debug implements Logger.
func (tl *testLogger) Debug(info, msg string) {
	tl.mux.Lock()
	defer tl.mux.Unlock()
	entry := fmt.Sprintf("%d [DEBUG] %s %s", time.Now().UnixNano(), info, msg)
	tl.entries = append(tl.entries, entry)
}

// Info implements Logger.
func (tl *testLogger) Info(info, msg string) {
	tl.mux.Lock()
	defer tl.mux.Unlock()
	entry := fmt.Sprintf("%d [INFO] %s %s", time.Now().UnixNano(), info, msg)
	tl.entries = append(tl.entries, entry)
}

// Warning implements Logger.
func (tl *testLogger) Warning(info, msg string) {
	tl.mux.Lock()
	defer tl.mux.Unlock()
	entry := fmt.Sprintf("%d [WARNING] %s %s", time.Now().UnixNano(), info, msg)
	tl.entries = append(tl.entries, entry)
}

// Error implements Logger.
func (tl *testLogger) Error(info, msg string) {
	tl.mux.Lock()
	defer tl.mux.Unlock()
	entry := fmt.Sprintf("%d [ERROR] %s %s", time.Now().UnixNano(), info, msg)
	tl.entries = append(tl.entries, entry)
}

// Critical implements Logger.
func (tl *testLogger) Critical(info, msg string) {
	tl.mux.Lock()
	defer tl.mux.Unlock()
	entry := fmt.Sprintf("%d [CRITICAL] %s %s", time.Now().UnixNano(), info, msg)
	tl.entries = append(tl.entries, entry)
}

// Fatal implements Logger.
func (tl *testLogger) Fatal(info, msg string) {
	tl.mux.Lock()
	defer tl.mux.Unlock()
	entry := fmt.Sprintf("%d [CRITICAL] %s %s", time.Now().UnixNano(), info, msg)
	tl.entries = append(tl.entries, entry)
}

// Len implements TestLogger.
func (tl *testLogger) Len() int {
	tl.mux.Lock()
	defer tl.mux.Unlock()
	return len(tl.entries)
}

// Entries implements TestLogger.
func (tl *testLogger) Entries() []string {
	tl.mux.Lock()
	defer tl.mux.Unlock()
	entries := make([]string, len(tl.entries))
	copy(entries, tl.entries)
	return entries
}

// Reset implements TestLogger.
func (tl *testLogger) Reset() {
	tl.mux.Lock()
	defer tl.mux.Unlock()
	tl.entries = nil
}

//--------------------
// HELPER
//--------------------

// callInfo bundles the info about the call environment
// when a logging statement occurred.
type callInfo struct {
	packageName string
	fileName    string
	funcName    string
	line        int
}

// shortFormat returns a string representation in a short variant.
func (ci *callInfo) shortFormat() string {
	return fmt.Sprintf("[%s]", ci.packageName)
}

// verboseFormat returns a string representation in a more verbose variant.
func (ci *callInfo) verboseFormat() string {
	return fmt.Sprintf("[%s] (%s:%s:%d)", ci.packageName, ci.fileName, ci.funcName, ci.line)
}

// retrieveCallInfo returns detailed information about
// the call location.
func retrieveCallInfo() *callInfo {
	pc, file, line, _ := runtime.Caller(2)
	_, fileName := path.Split(file)
	parts := strings.Split(runtime.FuncForPC(pc).Name(), ".")
	pl := len(parts)
	packageName := ""
	funcName := parts[pl-1]

	if parts[pl-2][0] == '(' {
		funcName = parts[pl-2] + "." + funcName
		packageName = strings.Join(parts[0:pl-2], ".")
	} else {
		packageName = strings.Join(parts[0:pl-1], ".")
	}

	return &callInfo{
		packageName: packageName,
		fileName:    fileName,
		funcName:    funcName,
		line:        line,
	}
}

// shallLog is used inside the logging functions to check if
// logging is wanted.
func shallLog(level LogLevel, info, msg string) bool {
	logMux.RLock()
	defer logMux.RUnlock()
	if logFilter == nil {
		return true
	}
	return !logFilter(level, info, msg)
}

// EOF
