// Tideland Go Library - Logger
//
// Copyright (C) 2012-2017 Frank Mueller / Tideland / Oldenburg / Germany
//
// All rights reserved. Use of this source code is governed
// by the new BSD license.

// Package logger of the Tideland Go Library provides a flexible way
// to log information with different levels and on different backends.
//
// The levels are Debug, Info, Warning, Error, Critical, and Fatal.
// Here logger.Debugf() also logs information about file name, function
// name, and line number while logger.Fatalf() may end the program
// depending on the set FatalExiterFunc.
//
// Different backends may be set. The standard logger writes to an
// io.Writer (initially os.Stdout), the go logger uses the Go log
// package, and the sys logger uses the Go syslog package on the
// according operating systems. For testing the test logger exists.
// When created also a fetch function is return. It returns the
// logged strings which can be used inside of tests then.
//
// Changes to the standard behavior can be made with logger.SetLevel(),
// logger.SetLogger(), and logger.SetFatalExiter(). Own logger
// backends and exiter can be defined.
package logger

// EOF
