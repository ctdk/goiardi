.. _logging:

Logging
=======

By default, goiardi logs to standard output and standard error. A log file may be specified with the ``-L/--log-file`` flag, or goiardi can log to syslog with the ``-s/--syslog`` flag on platforms that support syslog. Attempting to use syslog on a platform that doesn't support syslog (currently Windows and plan9 (although plan9 doesn't build for other reasons)) will result in an error.

Log levels
----------

Log levels can be set in goiardi with either the ``log-level`` option in the configuration file, the ``--log-level`` flag on the command line, the ``$GOIARDI_LOG_LEVEL`` environment variable, or with one to five ``-V`` flags on the command line. Log level options are "debug", "info", "warning", "error", "critical", and "fatal". More ``-V`` on the command line means more spewing into the log.

Policy logging
--------------

There is also a special flag for logging the RBAC & ACL operations. They have a separate special flag because of the **incredible** amount of output they spew into the logs, and absolutely swamp everything else. Since most of the time no one should need to use this, and because the casbin module that goiardi uses is what handles that log output, it's set separately than normal logging. 

It's described in the LINK TO docs/installation.rst page, but the flag is ``--policy-logging``, in the config file it's (unsurprisingly) ``policy-logging``, and the environment variable is ``GOIARDI_POLICY_LOGGING`` (which is also not all that surprising).
