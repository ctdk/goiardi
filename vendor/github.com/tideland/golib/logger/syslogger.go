// Tideland Go Library - Logger - SysLogger
//
// Copyright (C) 2012-2017 Frank Mueller / Tideland / Oldenburg / Germany
//
// All rights reserved. Use of this source code is governed
// by the new BSD license.

// +build !windows,!nacl,!plan9

package logger

//--------------------
// IMPORTS
//--------------------

import (
	"log"
	"log/syslog"
)

//--------------------
// SYSLOGGER
//--------------------

// SysLogger uses the Go syslog package as logging backend. It does
// not work on Windows or Plan9.
type SysLogger struct {
	writer *syslog.Writer
}

// NewSysLogger returns a logger implementation using the
// Go syslog package.
func NewSysLogger(tag string) (Logger, error) {
	writer, err := syslog.New(syslog.LOG_DEBUG|syslog.LOG_LOCAL0, tag)
	if err != nil {
		log.Fatalf("cannot init syslog: %v", err)
		return nil, err
	}
	return &SysLogger{writer}, nil
}

// Debug is specified on the Logger interface.
func (sl *SysLogger) Debug(info, msg string) {
	sl.writer.Debug(info + " " + msg)
}

// Info is specified on the Logger interface.
func (sl *SysLogger) Info(info, msg string) {
	sl.writer.Info(info + " " + msg)
}

// Warning is specified on the Logger interface.
func (sl *SysLogger) Warning(info, msg string) {
	sl.writer.Warning(info + " " + msg)
}

// Error is specified on the Logger interface.
func (sl *SysLogger) Error(info, msg string) {
	sl.writer.Err(info + " " + msg)
}

// Critical is specified on the Logger interface.
func (sl *SysLogger) Critical(info, msg string) {
	sl.writer.Crit(info + " " + msg)
}

// Fatal is specified on the Logger interface.
func (sl *SysLogger) Fatal(info, msg string) {
	sl.writer.Emerg(info + " " + msg)
}

// EOF
