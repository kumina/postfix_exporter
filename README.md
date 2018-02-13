# Prometheus Postfix exporter

This repository provides code for a Prometheus metrics exporter
for [the Postfix mail server](http://www.postfix.org/). This exporter
provides histogram metrics for the size and age of messages stored in
the mail queue. It extracts these metrics from Postfix by connecting to
a UNIX socket under `/var/spool`.

In addition to that, it counts events by parsing Postfix's log entries,
using regular expression matching.
The log entries are retrieved from the systemd journal or from a log file.

Please refer to this utility's `main()` function for a list of supported
command line flags.

## Events from log file

The log file is truncated when
processed, so that the next iteration doesn't interpret the same lines
twice. It makes sense to configure your syslogger to multiplex log
entries to a second file:

```
mail.* -/var/log/postfix_exporter_input.log
```

The path to the log file is specified with the `-postfix.logfile_path` flag.

## Events from systemd

Retrieval from the systemd journal is enabled with the `-systemd.enable` flag.
This overrides the log file setting.
It is possible to specify the unit (with `-systemd.unit`) or slice (with `-systemd.slice`).
Additionally, it is possible to read the journal from a directory with the `-systemd.journal_path` flag.
