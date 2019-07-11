# Prometheus Postfix exporter
[![CircleCI](https://circleci.com/gh/toliger/postfix_exporter.svg?style=svg)](https://circleci.com/gh/toliger/postfix_exporter)
[![Go Report Card](https://goreportcard.com/badge/github.com/toliger/postfix_exporter)](https://goreportcard.com/report/github.com/toliger/postfix_exporter)
[![GoDoc](https://godoc.org/github.com/toliger/postfix_exporter?status.svg)](https://godoc.org/github.com/toliger/postfix_exporter)
[![Docker Pulls](https://img.shields.io/docker/pulls/oligertimothee/postfix_exporter.svg?maxAge=604800)](https://hub.docker.com/r/oligertimothee/postfix_exporter/)

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

The log file is tailed when processed. Rotating the log files while the exporter
is running is OK. The path to the log file is specified with the
`--postfix.logfile_path` flag.

## Events from systemd

Retrieval from the systemd journal is enabled with the `--systemd.enable` flag.
This overrides the log file setting.
It is possible to specify the unit (with `--systemd.unit`) or slice (with `--systemd.slice`).
Additionally, it is possible to read the journal from a directory with the `--systemd.journal_path` flag.

## Build

You can easily build the app with:

```
make build
```
