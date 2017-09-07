# Prometheus Postfix exporter

This repository provides code for a [Prometheus](https://prometheus.io/) metrics exporter
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

# Usage

Command line arguments are:

```
  -postfix.logfile_path string
        Path where Postfix writes log entries. This file will be truncated by this exporter. (default "/var/log/postfix_exporter_input.log")
  -postfix.showq_path string
        Path at which Postfix places its showq socket. (default "/var/spool/postfix/public/showq")
  -web.listen-address string
        Address to listen on for web interface and telemetry. (default ":9154")
  -web.telemetry-path string
        Path under which to expose metrics. (default "/metrics")
```

## Docker Image

### Image Creation

You can create a Docker image by running the following:

```
make docker
```

This will create the image `postfix-exporter:latest`, which you can then [tag](https://docs.docker.com/engine/reference/commandline/tag/)
and [push](https://docs.docker.com/engine/reference/commandline/push/) appropriately to [Docker Hub](https://hub.docker.com).

### Running the Image

By default the image doesn't provide a [Postfix](https://postfix.org)
installation so you can test by using the host systems' Postfix UNIX
socket in the following manner:

```
docker run -it --rm \
    -v /var/spool/postfix/public/showq:/var/spool.postfix/public/showq \
    -p 9154:9154 \
    postfix-exporter:latest
```

Then you could be able to access the metrics from the host:

```
curl http://localhost:9154/metrics
```
