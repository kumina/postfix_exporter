// Copyright 2017 Kumina, https://kumina.nl/
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package logCollector

import (
	"context"
	"github.com/hpcloud/tail"
	"github.com/kumina/postfix_exporter/showq"
	"github.com/prometheus/client_golang/prometheus"
	"io"
	"log"
	"regexp"
	"strconv"
	"strings"
)

// LogCollector holds the state that should be preserved by the
// Postfix Prometheus metrics exporter across scrapes.
type LogCollector struct {
	journal             *Journal
	tailer              *tail.Tail
	logUnsupportedLines bool

	// Metrics that should persist after refreshes, based on logs.
	cleanupProcesses                prometheus.Counter
	cleanupRejects                  prometheus.Counter
	cleanupNotAccepted              prometheus.Counter
	lmtpDelays                      *prometheus.HistogramVec
	pipeDelays                      *prometheus.HistogramVec
	qmgrInsertsNrcpt                prometheus.Histogram
	qmgrInsertsSize                 prometheus.Histogram
	qmgrRemoves                     prometheus.Counter
	smtpDelays                      *prometheus.HistogramVec
	smtpTLSConnects                 *prometheus.CounterVec
	smtpConnectionTimedOut          prometheus.Counter
	smtpDeferreds                   prometheus.Counter
	smtpdConnects                   prometheus.Counter
	smtpdDisconnects                prometheus.Counter
	smtpdFCrDNSErrors               prometheus.Counter
	smtpdLostConnections            *prometheus.CounterVec
	smtpdProcesses                  *prometheus.CounterVec
	smtpdRejects                    *prometheus.CounterVec
	smtpdSASLAuthenticationFailures prometheus.Counter
	smtpdTLSConnects                *prometheus.CounterVec
	unsupportedLogEntries           *prometheus.CounterVec
	smtpStatusDeferred              prometheus.Counter
	opendkimSignatureAdded          *prometheus.CounterVec
	postfixUp                       showq.GaugeVec
}

// Patterns for parsing log messages.
var (
	logLine                             = regexp.MustCompile(` ?(postfix|opendkim)(/(\w+))?\[\d+\]: (.*)`)
	lmtpPipeSMTPLine                    = regexp.MustCompile(`, relay=(\S+), .*, delays=([0-9\.]+)/([0-9\.]+)/([0-9\.]+)/([0-9\.]+), `)
	qmgrInsertLine                      = regexp.MustCompile(`:.*, size=(\d+), nrcpt=(\d+) `)
	smtpStatusDeferredLine              = regexp.MustCompile(`, status=deferred`)
	smtpTLSLine                         = regexp.MustCompile(`^(\S+) TLS connection established to \S+: (\S+) with cipher (\S+) \((\d+)/(\d+) bits\)`)
	smtpConnectionTimedOut              = regexp.MustCompile(`^connect\s+to\s+(.*)\[(.*)\]:(\d+):\s+(Connection timed out)$`)
	smtpdFCrDNSErrorsLine               = regexp.MustCompile(`^warning: hostname \S+ does not resolve to address `)
	smtpdProcessesSASLLine              = regexp.MustCompile(`: client=.*, sasl_method=(\S+)`)
	smtpdRejectsLine                    = regexp.MustCompile(`^NOQUEUE: reject: RCPT from \S+: ([0-9]+) `)
	smtpdLostConnectionLine             = regexp.MustCompile(`^lost connection after (\w+) from `)
	smtpdSASLAuthenticationFailuresLine = regexp.MustCompile(`^warning: \S+: SASL \S+ authentication failed: `)
	smtpdTLSLine                        = regexp.MustCompile(`^(\S+) TLS connection established from \S+: (\S+) with cipher (\S+) \((\d+)/(\d+) bits\)$`)
	opendkimSignatureAdded              = regexp.MustCompile(`^[\w\d]+: DKIM-Signature field added \(s=(\w+), d=(.*)\)$`)
)

// CollectFromLogline collects metrict from a Postfix log line.
func (e *LogCollector) CollectFromLogLine(line string) {
	// Strip off timestamp, hostname, etc.
	logMatches := logLine.FindStringSubmatch(line)

	if logMatches == nil {
		// Unknown log entry format.
		e.addToUnsupportedLine(line, "")
		return
	}
	process := logMatches[1]
	remainder := logMatches[4]
	switch process {
	case "postfix":
		// Group patterns to check by Postfix service.
		subprocess := logMatches[3]
		switch subprocess {
		case "cleanup":
			if strings.Contains(remainder, ": message-id=<") {
				e.cleanupProcesses.Inc()
			} else if strings.Contains(remainder, ": reject: ") {
				e.cleanupRejects.Inc()
			} else {
				e.addToUnsupportedLine(line, subprocess)
			}
		case "lmtp":
			if lmtpMatches := lmtpPipeSMTPLine.FindStringSubmatch(remainder); lmtpMatches != nil {
				addToHistogramVec(e.lmtpDelays, lmtpMatches[2], "LMTP pdelay", "before_queue_manager")
				addToHistogramVec(e.lmtpDelays, lmtpMatches[3], "LMTP adelay", "queue_manager")
				addToHistogramVec(e.lmtpDelays, lmtpMatches[4], "LMTP sdelay", "connection_setup")
				addToHistogramVec(e.lmtpDelays, lmtpMatches[5], "LMTP xdelay", "transmission")
			} else {
				e.addToUnsupportedLine(line, subprocess)
			}
		case "pipe":
			if pipeMatches := lmtpPipeSMTPLine.FindStringSubmatch(remainder); pipeMatches != nil {
				addToHistogramVec(e.pipeDelays, pipeMatches[2], "PIPE pdelay", pipeMatches[1], "before_queue_manager")
				addToHistogramVec(e.pipeDelays, pipeMatches[3], "PIPE adelay", pipeMatches[1], "queue_manager")
				addToHistogramVec(e.pipeDelays, pipeMatches[4], "PIPE sdelay", pipeMatches[1], "connection_setup")
				addToHistogramVec(e.pipeDelays, pipeMatches[5], "PIPE xdelay", pipeMatches[1], "transmission")
			} else {
				e.addToUnsupportedLine(line, subprocess)
			}
		case "qmgr":
			if qmgrInsertMatches := qmgrInsertLine.FindStringSubmatch(remainder); qmgrInsertMatches != nil {
				addToHistogram(e.qmgrInsertsSize, qmgrInsertMatches[1], "QMGR size")
				addToHistogram(e.qmgrInsertsNrcpt, qmgrInsertMatches[2], "QMGR nrcpt")
			} else if strings.HasSuffix(remainder, ": removed") {
				e.qmgrRemoves.Inc()
			} else {
				e.addToUnsupportedLine(line, subprocess)
			}
		case "smtp":
			if smtpMatches := lmtpPipeSMTPLine.FindStringSubmatch(remainder); smtpMatches != nil {
				addToHistogramVec(e.smtpDelays, smtpMatches[2], "before_queue_manager", "")
				addToHistogramVec(e.smtpDelays, smtpMatches[3], "queue_manager", "")
				addToHistogramVec(e.smtpDelays, smtpMatches[4], "connection_setup", "")
				addToHistogramVec(e.smtpDelays, smtpMatches[5], "transmission", "")
				if smtpMatches := smtpStatusDeferredLine.FindStringSubmatch(remainder); smtpMatches != nil {
					e.smtpStatusDeferred.Inc()
				}
			} else if smtpTLSMatches := smtpTLSLine.FindStringSubmatch(remainder); smtpTLSMatches != nil {
				e.smtpTLSConnects.WithLabelValues(smtpTLSMatches[1:]...).Inc()
			} else if smtpMatches := smtpConnectionTimedOut.FindStringSubmatch(remainder); smtpMatches != nil {
				e.smtpConnectionTimedOut.Inc()
			} else {
				e.addToUnsupportedLine(line, subprocess)
			}
		case "smtpd":
			if strings.HasPrefix(remainder, "connect from ") {
				e.smtpdConnects.Inc()
			} else if strings.HasPrefix(remainder, "disconnect from ") {
				e.smtpdDisconnects.Inc()
			} else if smtpdFCrDNSErrorsLine.MatchString(remainder) {
				e.smtpdFCrDNSErrors.Inc()
			} else if smtpdLostConnectionMatches := smtpdLostConnectionLine.FindStringSubmatch(remainder); smtpdLostConnectionMatches != nil {
				e.smtpdLostConnections.WithLabelValues(smtpdLostConnectionMatches[1]).Inc()
			} else if smtpdProcessesSASLMatches := smtpdProcessesSASLLine.FindStringSubmatch(remainder); smtpdProcessesSASLMatches != nil {
				e.smtpdProcesses.WithLabelValues(smtpdProcessesSASLMatches[1]).Inc()
			} else if strings.Contains(remainder, ": client=") {
				e.smtpdProcesses.WithLabelValues("").Inc()
			} else if smtpdRejectsMatches := smtpdRejectsLine.FindStringSubmatch(remainder); smtpdRejectsMatches != nil {
				e.smtpdRejects.WithLabelValues(smtpdRejectsMatches[1]).Inc()
			} else if smtpdSASLAuthenticationFailuresLine.MatchString(remainder) {
				e.smtpdSASLAuthenticationFailures.Inc()
			} else if smtpdTLSMatches := smtpdTLSLine.FindStringSubmatch(remainder); smtpdTLSMatches != nil {
				e.smtpdTLSConnects.WithLabelValues(smtpdTLSMatches[1:]...).Inc()
			} else {
				e.addToUnsupportedLine(line, subprocess)
			}
		default:
			e.addToUnsupportedLine(line, subprocess)
		}
	case "opendkim":
		if opendkimMatches := opendkimSignatureAdded.FindStringSubmatch(remainder); opendkimMatches != nil {
			e.opendkimSignatureAdded.WithLabelValues(opendkimMatches[1], opendkimMatches[2]).Inc()
		} else {
			e.addToUnsupportedLine(line, process)
		}
	default:
		// Unknown log entry format.
		e.addToUnsupportedLine(line, "")
	}
}

func (e *LogCollector) addToUnsupportedLine(line string, subprocess string) {
	if e.logUnsupportedLines {
		log.Printf("Unsupported Line: %v", line)
	}
	e.unsupportedLogEntries.WithLabelValues(subprocess).Inc()
}

func addToHistogram(h prometheus.Histogram, value, fieldName string) {
	float, err := strconv.ParseFloat(value, 64)
	if err != nil {
		log.Printf("Couldn't convert value '%s' for %v: %v", value, fieldName, err)
	}
	h.Observe(float)
}
func addToHistogramVec(h *prometheus.HistogramVec, value, fieldName string, labels ...string) {
	float, err := strconv.ParseFloat(value, 64)
	if err != nil {
		log.Printf("Couldn't convert value '%s' for %v: %v", value, fieldName, err)
	}
	h.WithLabelValues(labels...).Observe(float)
}

// CollectLogfileFromFile tails a Postfix log file and collects entries from it.
func (e *LogCollector) CollectLogfileFromFile(ctx context.Context) {
	gauge := e.postfixUp.WithLabelValues(e.tailer.Filename)
	for {
		select {
		case line := <-e.tailer.Lines:
			e.CollectFromLogLine(line.Text)
		case <-ctx.Done():
			gauge.Set(0)
			return
		}
	}
}

// NewLogCollector creates a new Postfix exporter instance.
func NewLogCollector(logfilePath string, journal *Journal, logUnsupportedLines bool, postfixUp showq.GaugeVec) (*LogCollector, error) {
	var tailer *tail.Tail
	if logfilePath != "" {
		var err error
		tailer, err = tail.TailFile(logfilePath, tail.Config{
			ReOpen:    true,                               // reopen the file if it's rotated
			MustExist: true,                               // fail immediately if the file is missing or has incorrect permissions
			Follow:    true,                               // run in follow mode
			Location:  &tail.SeekInfo{Whence: io.SeekEnd}, // seek to end of file
		})
		if err != nil {
			return nil, err
		}
	}
	timeBuckets := []float64{1e-3, 1e-2, 1e-1, 1.0, 10, 1 * 60, 1 * 60 * 60, 24 * 60 * 60, 2 * 24 * 60 * 60}
	return &LogCollector{
		logUnsupportedLines: logUnsupportedLines,
		tailer:              tailer,
		journal:             journal,

		postfixUp: postfixUp,
		cleanupProcesses: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "postfix",
			Name:      "cleanup_messages_processed_total",
			Help:      "Total number of messages processed by cleanup.",
		}),
		cleanupRejects: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "postfix",
			Name:      "cleanup_messages_rejected_total",
			Help:      "Total number of messages rejected by cleanup.",
		}),
		cleanupNotAccepted: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "postfix",
			Name:      "cleanup_messages_not_accepted_total",
			Help:      "Total number of messages not accepted by cleanup.",
		}),
		lmtpDelays: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "postfix",
				Name:      "lmtp_delivery_delay_seconds",
				Help:      "LMTP message processing time in seconds.",
				Buckets:   timeBuckets,
			},
			[]string{"stage"}),
		pipeDelays: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "postfix",
				Name:      "pipe_delivery_delay_seconds",
				Help:      "Pipe message processing time in seconds.",
				Buckets:   timeBuckets,
			},
			[]string{"relay", "stage"}),
		qmgrInsertsNrcpt: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: "postfix",
			Name:      "qmgr_messages_inserted_receipients",
			Help:      "Number of receipients per message inserted into the mail queues.",
			Buckets:   []float64{1, 2, 4, 8, 16, 32, 64, 128},
		}),
		qmgrInsertsSize: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: "postfix",
			Name:      "qmgr_messages_inserted_size_bytes",
			Help:      "Size of messages inserted into the mail queues in bytes.",
			Buckets:   []float64{1e3, 1e4, 1e5, 1e6, 1e7, 1e8, 1e9},
		}),
		qmgrRemoves: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "postfix",
			Name:      "qmgr_messages_removed_total",
			Help:      "Total number of messages removed from mail queues.",
		}),
		smtpDelays: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "postfix",
				Name:      "smtp_delivery_delay_seconds",
				Help:      "SMTP message processing time in seconds.",
				Buckets:   timeBuckets,
			},
			[]string{"stage"}),
		smtpTLSConnects: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "postfix",
				Name:      "smtp_tls_connections_total",
				Help:      "Total number of outgoing TLS connections.",
			},
			[]string{"trust", "protocol", "cipher", "secret_bits", "algorithm_bits"}),
		smtpDeferreds: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "postfix",
			Name:      "smtp_deferred_messages_total",
			Help:      "Total number of messages that have been deferred on SMTP.",
		}),
		smtpConnectionTimedOut: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "postfix",
			Name:      "smtp_connection_timed_out_total",
			Help:      "Total number of messages that have been deferred on SMTP.",
		}),
		smtpdConnects: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "postfix",
			Name:      "smtpd_connects_total",
			Help:      "Total number of incoming connections.",
		}),
		smtpdDisconnects: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "postfix",
			Name:      "smtpd_disconnects_total",
			Help:      "Total number of incoming disconnections.",
		}),
		smtpdFCrDNSErrors: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "postfix",
			Name:      "smtpd_forward_confirmed_reverse_dns_errors_total",
			Help:      "Total number of connections for which forward-confirmed DNS cannot be resolved.",
		}),
		smtpdLostConnections: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "postfix",
				Name:      "smtpd_connections_lost_total",
				Help:      "Total number of connections lost.",
			},
			[]string{"after_stage"}),
		smtpdProcesses: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "postfix",
				Name:      "smtpd_messages_processed_total",
				Help:      "Total number of messages processed.",
			},
			[]string{"sasl_method"}),
		smtpdRejects: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "postfix",
				Name:      "smtpd_messages_rejected_total",
				Help:      "Total number of NOQUEUE rejects.",
			},
			[]string{"code"}),
		smtpdSASLAuthenticationFailures: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "postfix",
			Name:      "smtpd_sasl_authentication_failures_total",
			Help:      "Total number of SASL authentication failures.",
		}),
		smtpdTLSConnects: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "postfix",
				Name:      "smtpd_tls_connections_total",
				Help:      "Total number of incoming TLS connections.",
			},
			[]string{"trust", "protocol", "cipher", "secret_bits", "algorithm_bits"}),
		unsupportedLogEntries: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "postfix",
				Name:      "unsupported_log_entries_total",
				Help:      "Log entries that could not be processed.",
			},
			[]string{"service"}),
		smtpStatusDeferred: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "postfix",
			Name:      "smtp_status_deferred",
			Help:      "Total number of messages deferred.",
		}),
		opendkimSignatureAdded: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "opendkim",
				Name:      "signatures_added_total",
				Help:      "Total number of messages signed.",
			},
			[]string{"subject", "domain"},
		),
	}, nil
}

func (e *LogCollector) foreverCollectFromJournal(ctx context.Context) {
	select {
	case <-ctx.Done():
		e.postfixUp.WithLabelValues(e.journal.Path).Set(0)
		return
	default:
		err := e.CollectLogfileFromJournal()
		if err != nil {
			log.Printf("Couldn't read journal: %v", err)
			e.postfixUp.WithLabelValues(e.journal.Path).Set(0)
		} else {
			e.postfixUp.WithLabelValues(e.journal.Path).Set(1)
		}
	}
}

func (e *LogCollector) StartMetricCollection(ctx context.Context) <-chan interface{} {
	done := make(chan interface{})
	go func() {
		defer close(done)
		if e.journal != nil {
			e.foreverCollectFromJournal(ctx)
		} else {
			e.CollectLogfileFromFile(ctx)
		}
	}()
	return done
}

// Describe the Prometheus metrics that are going to be exported.
func (e *LogCollector) Describe(ch chan<- *prometheus.Desc) {
	e.cleanupProcesses.Describe(ch)
	e.cleanupRejects.Describe(ch)
	e.cleanupNotAccepted.Describe(ch)
	e.lmtpDelays.Describe(ch)
	e.pipeDelays.Describe(ch)
	e.qmgrInsertsNrcpt.Describe(ch)
	e.qmgrInsertsSize.Describe(ch)
	e.qmgrRemoves.Describe(ch)
	e.smtpDelays.Describe(ch)
	e.smtpTLSConnects.Describe(ch)
	e.smtpDeferreds.Describe(ch)
	e.smtpdConnects.Describe(ch)
	e.smtpdDisconnects.Describe(ch)
	e.smtpdFCrDNSErrors.Describe(ch)
	e.smtpdLostConnections.Describe(ch)
	e.smtpdProcesses.Describe(ch)
	e.smtpdRejects.Describe(ch)
	e.smtpdSASLAuthenticationFailures.Describe(ch)
	e.smtpdTLSConnects.Describe(ch)
	e.smtpStatusDeferred.Describe(ch)
	e.unsupportedLogEntries.Describe(ch)
	e.smtpConnectionTimedOut.Describe(ch)
	e.opendkimSignatureAdded.Describe(ch)
}

// Collect metrics from Postfix's showq socket and its log file.
func (e *LogCollector) Collect(ch chan<- prometheus.Metric) {
	e.cleanupProcesses.Collect(ch)
	e.cleanupRejects.Collect(ch)
	e.cleanupNotAccepted.Collect(ch)
	e.lmtpDelays.Collect(ch)
	e.pipeDelays.Collect(ch)
	e.qmgrInsertsNrcpt.Collect(ch)
	e.qmgrInsertsSize.Collect(ch)
	e.qmgrRemoves.Collect(ch)
	e.smtpDelays.Collect(ch)
	e.smtpTLSConnects.Collect(ch)
	e.smtpDeferreds.Collect(ch)
	e.smtpdConnects.Collect(ch)
	e.smtpdDisconnects.Collect(ch)
	e.smtpdFCrDNSErrors.Collect(ch)
	e.smtpdLostConnections.Collect(ch)
	e.smtpdProcesses.Collect(ch)
	e.smtpdRejects.Collect(ch)
	e.smtpdSASLAuthenticationFailures.Collect(ch)
	e.smtpdTLSConnects.Collect(ch)
	e.smtpStatusDeferred.Collect(ch)
	e.unsupportedLogEntries.Collect(ch)
	e.smtpConnectionTimedOut.Collect(ch)
	e.opendkimSignatureAdded.Collect(ch)
}
