.. _logging:

Logging
=======

By default, goiardi logs to standard output and standard error. A log file may be specified with the ``-L/--log-file`` flag, or goiardi can log to syslog with the ``-s/--syslog`` flag on platforms that support syslog. Attempting to use syslog on a platform that doesn't support syslog (currently Windows and plan9 (although plan9 doesn't build for other reasons)) will result in an error.

Log levels
----------

Log levels can be set in goiardi with either the ``log-level`` option in the configuration file, the ``--log-level`` flag on the command line, the ``$GOIARDI_LOG_LEVEL`` environment variable, or with one to five ``-V`` flags on the command line. Log level options are "debug", "info", "warning", "error", "critical", and "fatal". More ``-V`` on the command line means more spewing into the log.
