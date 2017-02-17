# Prometheus Postfix exporter

This repository provides code for a simple Prometheus metrics exporter
for [the Postfix mail server](http://www.postfix.org/). Right now, this
exporter only provides histogram metrics for the size and age of
messages stored in the mail queue. It extracts these metrics from
Postfix by connecting to a UNIX socket under `/var/spool`.

Please refer to this utility's `main()` function for a list of supported
command line flags.
