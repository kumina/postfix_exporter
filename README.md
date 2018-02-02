# Prometheus Postfix exporter

This repository provides code for a Prometheus metrics exporter
for [the Postfix mail server](http://www.postfix.org/). This exporter
provides histogram metrics for the size and age of messages stored in
the mail queue. It extracts these metrics from Postfix by connecting to
a UNIX socket under `/var/spool`.

In addition to that, it counts events by parsing Postfix's log file,
using regular expression matching. It truncates the log file when
processed, so that the next iteration doesn't interpret the same lines
twice. It makes sense to configure your syslogger to multiplex log
entries to a second file:

```
mail.* -/var/log/postfix_exporter_input.log
```

There is also an option to collect the metrics via the systemd journal instead of a log file.


Please refer to this utility's `main()` function for a list of supported
command line flags.
