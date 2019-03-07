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

package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"github.com/alecthomas/kingpin"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/hpcloud/tail"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	postfixUpDesc = prometheus.NewDesc(
		prometheus.BuildFQName("postfix", "", "up"),
		"Whether scraping Postfix's metrics was successful.",
		[]string{"path"}, nil)
)

// PostfixExporter holds the state that should be preserved by the
// Postfix Prometheus metrics exporter across scrapes.
type PostfixExporter struct {
	showqPath string
	journal   *Journal
	tailer    *tail.Tail

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
}

// CollectShowqFromReader parses the output of Postfix's 'showq' command
// and turns it into metrics.
//
// The output format of this command depends on the version of Postfix
// used. Postfix 2.x uses a textual format, identical to the output of
// the 'mailq' command. Postfix 3.x uses a binary format, where entries
// are terminated using null bytes. Auto-detect the format by scanning
// for null bytes in the first 128 bytes of output.
func CollectShowqFromReader(file io.Reader, ch chan<- prometheus.Metric) error {
	reader := bufio.NewReader(file)
	buf, err := reader.Peek(128)
	if err != nil && err != io.EOF {
		log.Printf("Could not read postfix output, %v", err)
	}
	if bytes.IndexByte(buf, 0) >= 0 {
		return CollectBinaryShowqFromReader(reader, ch)
	}
	return CollectTextualShowqFromReader(reader, ch)
}

// CollectTextualShowqFromReader parses Postfix's textual showq output.
func CollectTextualShowqFromReader(file io.Reader, ch chan<- prometheus.Metric) error {
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)

	// Regular expression for matching postqueue's output. Example:
	// "A07A81514      5156 Tue Feb 14 13:13:54  MAILER-DAEMON"
	messageLine := regexp.MustCompile(`^[0-9A-F]+([\*!]?) +(\d+) (\w{3} \w{3} +\d+ +\d+:\d{2}:\d{2}) +`)

	// Histograms tracking the messages by size and age.
	sizeHistogram := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "postfix",
			Name:      "showq_message_size_bytes",
			Help:      "Size of messages in Postfix's message queue, in bytes",
			Buckets:   []float64{1e3, 1e4, 1e5, 1e6, 1e7, 1e8, 1e9},
		},
		[]string{"queue"})
	ageHistogram := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "postfix",
			Name:      "showq_message_age_seconds",
			Help:      "Age of messages in Postfix's message queue, in seconds",
			Buckets:   []float64{1e1, 1e2, 1e3, 1e4, 1e5, 1e6, 1e7, 1e8},
		},
		[]string{"queue"})

	// Initialize all queue buckets to zero.
	for _, q := range []string{"active", "hold", "other"} {
		sizeHistogram.WithLabelValues(q)
		ageHistogram.WithLabelValues(q)
	}

	now := time.Now()
	location, err := time.LoadLocation("Local")
	if err != nil {
		log.Println(err)
	}

	for scanner.Scan() {
		matches := messageLine.FindStringSubmatch(scanner.Text())
		if matches != nil {
			// Derive the name of the message queue.
			queue := "other"
			if matches[1] == "*" {
				queue = "active"
			} else if matches[1] == "!" {
				queue = "hold"
			}

			// Parse the message size.
			size, err := strconv.ParseFloat(matches[2], 64)
			if err != nil {
				return err
			}

			// Parse the message date. Unfortunately, the
			// output contains no year number. Assume it
			// applies to the last year for which the
			// message date doesn't exceed time.Now().
			date, err := time.ParseInLocation("Mon Jan 2 15:04:05",
				matches[3], location)
			if err != nil {
				return err
			}
			date = date.AddDate(now.Year(), 0, 0)
			if date.After(now) {
				date = date.AddDate(-1, 0, 0)
			}

			sizeHistogram.WithLabelValues(queue).Observe(size)
			ageHistogram.WithLabelValues(queue).Observe(now.Sub(date).Seconds())
		}
	}

	sizeHistogram.Collect(ch)
	ageHistogram.Collect(ch)
	return scanner.Err()
}

// ScanNullTerminatedEntries is a splitting function for bufio.Scanner
// to split entries by null bytes.
func ScanNullTerminatedEntries(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if i := bytes.IndexByte(data, 0); i >= 0 {
		// Valid record found.
		return i + 1, data[0:i], nil
	} else if atEOF && len(data) != 0 {
		// Data at the end of the file without a null terminator.
		return 0, nil, errors.New("Expected null byte terminator")
	} else {
		// Request more data.
		return 0, nil, nil
	}
}

// CollectBinaryShowqFromReader parses Postfix's binary showq format.
func CollectBinaryShowqFromReader(file io.Reader, ch chan<- prometheus.Metric) error {
	scanner := bufio.NewScanner(file)
	scanner.Split(ScanNullTerminatedEntries)

	// Histograms tracking the messages by size and age.
	sizeHistogram := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "postfix",
			Name:      "showq_message_size_bytes",
			Help:      "Size of messages in Postfix's message queue, in bytes",
			Buckets:   []float64{1e3, 1e4, 1e5, 1e6, 1e7, 1e8, 1e9},
		},
		[]string{"queue"})
	ageHistogram := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "postfix",
			Name:      "showq_message_age_seconds",
			Help:      "Age of messages in Postfix's message queue, in seconds",
			Buckets:   []float64{1e1, 1e2, 1e3, 1e4, 1e5, 1e6, 1e7, 1e8},
		},
		[]string{"queue"})

	// Initialize all queue buckets to zero.
	for _, q := range []string{"active", "deferred", "hold", "incoming", "maildrop"} {
		sizeHistogram.WithLabelValues(q)
		ageHistogram.WithLabelValues(q)
	}

	now := float64(time.Now().UnixNano()) / 1e9
	queue := "unknown"
	for scanner.Scan() {
		// Parse a key/value entry.
		key := scanner.Text()
		if len(key) == 0 {
			// Empty key means a record separator.
			queue = "unknown"
			continue
		}
		if !scanner.Scan() {
			return fmt.Errorf("key %q does not have a value", key)
		}
		value := scanner.Text()

		if key == "queue_name" {
			// The name of the message queue.
			queue = value
		} else if key == "size" {
			// Message size in bytes.
			size, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return err
			}
			sizeHistogram.WithLabelValues(queue).Observe(size)
		} else if key == "time" {
			// Message time as a UNIX timestamp.
			utime, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return err
			}
			ageHistogram.WithLabelValues(queue).Observe(now - utime)
		}
	}

	sizeHistogram.Collect(ch)
	ageHistogram.Collect(ch)
	return scanner.Err()
}

// CollectShowqFromFile collects Postfix queue statistics from a file.
//func CollectShowqFromFile(path string, ch chan<- prometheus.Metric) error {
//	fd, err := os.Open(path)
//	if err != nil {
//		return err
//	}
//	defer fd.Close()
//	return CollectShowqFromReader(fd, ch)
//}

// CollectShowqFromSocket collects Postfix queue statistics from a socket.
func CollectShowqFromSocket(path string, ch chan<- prometheus.Metric) error {
	fd, err := net.Dial("unix", path)
	if err != nil {
		return err
	}
	defer fd.Close()
	return CollectShowqFromReader(fd, ch)
}

// Patterns for parsing log messages.
var (
	logLine                             = regexp.MustCompile(` ?postfix/(\w+)\[\d+\]: (.*)`)
	lmtpPipeSMTPLine                    = regexp.MustCompile(`, relay=(\S+), .*, delays=([0-9\.]+)/([0-9\.]+)/([0-9\.]+)/([0-9\.]+), `)
	qmgrInsertLine                      = regexp.MustCompile(`:.*, size=(\d+), nrcpt=(\d+) `)
	smtpTLSLine                         = regexp.MustCompile(`^(\S+) TLS connection established to \S+: (\S+) with cipher (\S+) \((\d+)/(\d+) bits\)$`)
	smtpDeferredsLine                   = regexp.MustCompile(`status=deferred`)
	smtpdFCrDNSErrorsLine               = regexp.MustCompile(`^warning: hostname \S+ does not resolve to address `)
	smtpdProcessesSASLLine              = regexp.MustCompile(`: client=.*, sasl_username=(\S+)`)
	smtpdRejectsLine                    = regexp.MustCompile(`^NOQUEUE: reject: RCPT from \S+: ([0-9]+) `)
	smtpdLostConnectionLine             = regexp.MustCompile(`^lost connection after (\w+) from `)
	smtpdSASLAuthenticationFailuresLine = regexp.MustCompile(`^warning: \S+: SASL \S+ authentication failed: `)
	smtpdTLSLine                        = regexp.MustCompile(`^(\S+) TLS connection established from \S+: (\S+) with cipher (\S+) \((\d+)/(\d+) bits\)$`)
)

// CollectFromLogline collects metrict from a Postfix log line.
func (e *PostfixExporter) CollectFromLogline(line string) {
	// Strip off timestamp, hostname, etc.
	if logMatches := logLine.FindStringSubmatch(line); logMatches != nil {
		// Group patterns to check by Postfix service.
		if logMatches[1] == "cleanup" {
			if strings.Contains(logMatches[2], ": message-id=<") {
				e.cleanupProcesses.Inc()
			} else if strings.Contains(logMatches[2], ": reject: ") {
				e.cleanupRejects.Inc()
			} else if strings.Contains(logMatches[2], "message not accepted") {
				e.cleanupNotAccepted.Inc()
			} else {
				e.unsupportedLogEntries.WithLabelValues(logMatches[1]).Inc()
			}
		} else if logMatches[1] == "lmtp" {
			if lmtpMatches := lmtpPipeSMTPLine.FindStringSubmatch(logMatches[2]); lmtpMatches != nil {
				pdelay, err := strconv.ParseFloat(lmtpMatches[2], 64)
				if err != nil {
					log.Printf("Couldn't convert LMTP pdelay: %v", err)
				}
				e.lmtpDelays.WithLabelValues("before_queue_manager").Observe(pdelay)
				adelay, err := strconv.ParseFloat(lmtpMatches[3], 64)
				if err != nil {
					log.Printf("Couldn't convert LMTP adelay: %v", err)
				}
				e.lmtpDelays.WithLabelValues("queue_manager").Observe(adelay)
				sdelay, err := strconv.ParseFloat(lmtpMatches[4], 64)
				if err != nil {
					log.Printf("Couldn't convert LMTP adelay: %v", err)
				}
				e.lmtpDelays.WithLabelValues("connection_setup").Observe(sdelay)
				xdelay, err := strconv.ParseFloat(lmtpMatches[5], 64)
				if err != nil {
					log.Printf("Couldn't convert LMTP xdelay: %v", err)
				}
				e.lmtpDelays.WithLabelValues("transmission").Observe(xdelay)
			} else {
				e.unsupportedLogEntries.WithLabelValues(logMatches[1]).Inc()
			}
		} else if logMatches[1] == "pipe" {
			if pipeMatches := lmtpPipeSMTPLine.FindStringSubmatch(logMatches[2]); pipeMatches != nil {
				pdelay, err := strconv.ParseFloat(pipeMatches[2], 64)
				if err != nil {
					log.Printf("Couldn't convert PIPE pdelay: %v", err)
				}
				e.pipeDelays.WithLabelValues(pipeMatches[1], "before_queue_manager").Observe(pdelay)
				adelay, err := strconv.ParseFloat(pipeMatches[3], 64)
				if err != nil {
					log.Printf("Couldn't convert PIPE adelay: %v", err)
				}
				e.pipeDelays.WithLabelValues(pipeMatches[1], "queue_manager").Observe(adelay)
				sdelay, err := strconv.ParseFloat(pipeMatches[4], 64)
				if err != nil {
					log.Printf("Couldn't convert PIPE sdelay: %v", err)
				}
				e.pipeDelays.WithLabelValues(pipeMatches[1], "connection_setup").Observe(sdelay)
				xdelay, err := strconv.ParseFloat(pipeMatches[5], 64)
				if err != nil {
					log.Printf("Couldn't convert PIPE xdelay: %v", err)
				}
				e.pipeDelays.WithLabelValues(pipeMatches[1], "transmission").Observe(xdelay)
			} else {
				e.unsupportedLogEntries.WithLabelValues(logMatches[1]).Inc()
			}
		} else if logMatches[1] == "qmgr" {
			if qmgrInsertMatches := qmgrInsertLine.FindStringSubmatch(logMatches[2]); qmgrInsertMatches != nil {
				size, err := strconv.ParseFloat(qmgrInsertMatches[1], 64)
				if err != nil {
					log.Printf("Couldn't convert QMGR size: %v", err)
				}
				e.qmgrInsertsSize.Observe(size)
				nrcpt, err := strconv.ParseFloat(qmgrInsertMatches[2], 64)
				if err != nil {
					log.Printf("Couldn't convert QMGR nrcpt: %v", err)
				}
				e.qmgrInsertsNrcpt.Observe(nrcpt)
			} else if strings.HasSuffix(logMatches[2], ": removed") {
				e.qmgrRemoves.Inc()
			} else {
				e.unsupportedLogEntries.WithLabelValues(logMatches[1]).Inc()
			}
		} else if logMatches[1] == "smtp" {
			if smtpMatches := lmtpPipeSMTPLine.FindStringSubmatch(logMatches[2]); smtpMatches != nil {
				pdelay, err := strconv.ParseFloat(smtpMatches[2], 64)
				if err != nil {
					log.Printf("Couldn't convert SMTP pdelay: %v", err)
				}
				e.smtpDelays.WithLabelValues("before_queue_manager").Observe(pdelay)
				adelay, err := strconv.ParseFloat(smtpMatches[3], 64)
				if err != nil {
					log.Printf("Couldn't convert SMTP adelay: %v", err)
				}
				e.smtpDelays.WithLabelValues("queue_manager").Observe(adelay)
				sdelay, err := strconv.ParseFloat(smtpMatches[4], 64)
				if err != nil {
					log.Printf("Couldn't convert SMTP sdelay: %v", err)
				}
				e.smtpDelays.WithLabelValues("connection_setup").Observe(sdelay)
				xdelay, err := strconv.ParseFloat(smtpMatches[5], 64)
				if err != nil {
					log.Printf("Couldn't convert SMTP xdelay: %v", err)
				}
				e.smtpDelays.WithLabelValues("transmission").Observe(xdelay)
				if smtpDeferredsMatches := smtpDeferredsLine.FindStringSubmatch(
					logMatches[2]); smtpDeferredsMatches != nil {
					e.smtpDeferreds.Inc()
				}
			} else if smtpTLSMatches := smtpTLSLine.FindStringSubmatch(logMatches[2]); smtpTLSMatches != nil {
				e.smtpTLSConnects.WithLabelValues(smtpTLSMatches[1:]...).Inc()
			} else {
				e.unsupportedLogEntries.WithLabelValues(logMatches[1]).Inc()
			}
		} else if logMatches[1] == "smtpd" {
			if strings.HasPrefix(logMatches[2], "connect from ") {
				e.smtpdConnects.Inc()
			} else if strings.HasPrefix(logMatches[2], "disconnect from ") {
				e.smtpdDisconnects.Inc()
			} else if smtpdFCrDNSErrorsLine.MatchString(logMatches[2]) {
				e.smtpdFCrDNSErrors.Inc()
			} else if smtpdLostConnectionMatches := smtpdLostConnectionLine.FindStringSubmatch(logMatches[2]); smtpdLostConnectionMatches != nil {
				e.smtpdLostConnections.WithLabelValues(smtpdLostConnectionMatches[1]).Inc()
			} else if smtpdProcessesSASLMatches := smtpdProcessesSASLLine.FindStringSubmatch(logMatches[2]); smtpdProcessesSASLMatches != nil {
				e.smtpdProcesses.WithLabelValues(smtpdProcessesSASLMatches[1]).Inc()
			} else if strings.Contains(logMatches[2], ": client=") {
				e.smtpdProcesses.WithLabelValues("").Inc()
			} else if smtpdRejectsMatches := smtpdRejectsLine.FindStringSubmatch(logMatches[2]); smtpdRejectsMatches != nil {
				e.smtpdRejects.WithLabelValues(smtpdRejectsMatches[1]).Inc()
			} else if smtpdSASLAuthenticationFailuresLine.MatchString(logMatches[2]) {
				e.smtpdSASLAuthenticationFailures.Inc()
			} else if smtpdTLSMatches := smtpdTLSLine.FindStringSubmatch(logMatches[2]); smtpdTLSMatches != nil {
				e.smtpdTLSConnects.WithLabelValues(smtpdTLSMatches[1:]...).Inc()
			} else {
				e.unsupportedLogEntries.WithLabelValues(logMatches[1]).Inc()
			}
		} else {
			// Unknown Postfix service.
			e.unsupportedLogEntries.WithLabelValues(logMatches[1]).Inc()
		}
	} else {
		// Unknown log entry format.
		e.unsupportedLogEntries.WithLabelValues("").Inc()
	}
}

// CollectLogfileFromFile tails a Postfix log file and collects entries from it.
func (e *PostfixExporter) CollectLogfileFromFile() error {
	for {
		select {
		case line := <-e.tailer.Lines:
			e.CollectFromLogline(line.Text)
		default:
			return nil
		}
	}
}

// NewPostfixExporter creates a new Postfix exporter instance.
func NewPostfixExporter(showqPath string, logfilePath string, journal *Journal) (*PostfixExporter, error) {
	var tailer *tail.Tail
	if logfilePath != "" {
		var err error
		tailer, err = tail.TailFile(logfilePath, tail.Config{
			ReOpen:    true, // reopen the file if it's rotated
			MustExist: true, // fail immediately if the file is missing or has incorrect permissions
			Follow:    true, // run in follow mode
		})
		if err != nil {
			return nil, err
		}
	}
	return &PostfixExporter{
		showqPath: showqPath,
		tailer:    tailer,
		journal:   journal,

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
				Buckets:   []float64{1e-3, 1e-2, 1e-1, 1e0, 1e1, 1e2, 1e3},
			},
			[]string{"stage"}),
		pipeDelays: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "postfix",
				Name:      "pipe_delivery_delay_seconds",
				Help:      "Pipe message processing time in seconds.",
				Buckets:   []float64{1e-3, 1e-2, 1e-1, 1e0, 1e1, 1e2, 1e3},
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
				Buckets:   []float64{1e-3, 1e-2, 1e-1, 1e0, 1e1, 1e2, 1e3},
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
			[]string{"sasl_username"}),
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
	}, nil
}

// Describe the Prometheus metrics that are going to be exported.
func (e *PostfixExporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- postfixUpDesc

	ch <- e.cleanupProcesses.Desc()
	ch <- e.cleanupRejects.Desc()
	ch <- e.cleanupNotAccepted.Desc()
	e.lmtpDelays.Describe(ch)
	e.pipeDelays.Describe(ch)
	ch <- e.qmgrInsertsNrcpt.Desc()
	ch <- e.qmgrInsertsSize.Desc()
	ch <- e.qmgrRemoves.Desc()
	e.smtpDelays.Describe(ch)
	e.smtpTLSConnects.Describe(ch)
	ch <- e.smtpDeferreds.Desc()
	ch <- e.smtpdConnects.Desc()
	ch <- e.smtpdDisconnects.Desc()
	ch <- e.smtpdFCrDNSErrors.Desc()
	e.smtpdLostConnections.Describe(ch)
	e.smtpdProcesses.Describe(ch)
	e.smtpdRejects.Describe(ch)
	ch <- e.smtpdSASLAuthenticationFailures.Desc()
	e.smtpdTLSConnects.Describe(ch)
	e.unsupportedLogEntries.Describe(ch)
}

// Collect metrics from Postfix's showq socket and its log file.
func (e *PostfixExporter) Collect(ch chan<- prometheus.Metric) {
	err := CollectShowqFromSocket(e.showqPath, ch)
	if err == nil {
		ch <- prometheus.MustNewConstMetric(
			postfixUpDesc,
			prometheus.GaugeValue,
			1.0,
			e.showqPath)
	} else {
		log.Printf("Failed to scrape showq socket: %s", err)
		ch <- prometheus.MustNewConstMetric(
			postfixUpDesc,
			prometheus.GaugeValue,
			0.0,
			e.showqPath)
	}

	var src string
	if e.journal != nil {
		err = e.CollectLogfileFromJournal()
		src = e.journal.Path
	} else {
		err = e.CollectLogfileFromFile()
		src = e.tailer.Filename
	}
	if err == nil {
		ch <- prometheus.MustNewConstMetric(
			postfixUpDesc,
			prometheus.GaugeValue,
			1.0,
			src)
	} else {
		log.Printf("Failed to scrape log: %s", err)
		ch <- prometheus.MustNewConstMetric(
			postfixUpDesc,
			prometheus.GaugeValue,
			0.0,
			src)
	}

	ch <- e.cleanupProcesses
	ch <- e.cleanupRejects
	ch <- e.cleanupNotAccepted
	e.lmtpDelays.Collect(ch)
	e.pipeDelays.Collect(ch)
	ch <- e.qmgrInsertsNrcpt
	ch <- e.qmgrInsertsSize
	ch <- e.qmgrRemoves
	e.smtpDelays.Collect(ch)
	e.smtpTLSConnects.Collect(ch)
	ch <- e.smtpDeferreds
	ch <- e.smtpdConnects
	ch <- e.smtpdDisconnects
	ch <- e.smtpdFCrDNSErrors
	e.smtpdLostConnections.Collect(ch)
	e.smtpdProcesses.Collect(ch)
	e.smtpdRejects.Collect(ch)
	ch <- e.smtpdSASLAuthenticationFailures
	e.smtpdTLSConnects.Collect(ch)
	e.unsupportedLogEntries.Collect(ch)
}

func main() {
	var (
		app                                           = kingpin.New("postfix_exporter", "Prometheus metrics exporter for postfix")
		listenAddress                                 = app.Flag("web.listen-address", "Address to listen on for web interface and telemetry.").Default(":9154").String()
		metricsPath                                   = app.Flag("web.telemetry-path", "Path under which to expose metrics.").Default("/metrics").String()
		postfixShowqPath                              = app.Flag("postfix.showq_path", "Path at which Postfix places its showq socket.").Default("/var/spool/postfix/public/showq").String()
		postfixLogfilePath                            = app.Flag("postfix.logfile_path", "Path where Postfix writes log entries. This file will be truncated by this exporter.").Default("/var/log/postfix_exporter_input.log").String()
		systemdEnable                                 bool
		systemdUnit, systemdSlice, systemdJournalPath string
	)
	systemdFlags(&systemdEnable, &systemdUnit, &systemdSlice, &systemdJournalPath, app)

	kingpin.MustParse(app.Parse(os.Args[1:]))

	var journal *Journal
	if systemdEnable {
		var err error
		journal, err = NewJournal(systemdUnit, systemdSlice, systemdJournalPath)
		if err != nil {
			log.Fatalf("Error opening systemd journal: %s", err)
		}
		defer journal.Close()
	}

	exporter, err := NewPostfixExporter(
		*postfixShowqPath,
		*postfixLogfilePath,
		journal,
	)
	if err != nil {
		log.Fatalf("Failed to create PostfixExporter: %s", err)
	}
	prometheus.MustRegister(exporter)

	http.Handle(*metricsPath, prometheus.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, err = w.Write([]byte(`
			<html>
			<head><title>Postfix Exporter</title></head>
			<body>
			<h1>Postfix Exporter</h1>
			<p><a href='` + *metricsPath + `'>Metrics</a></p>
			</body>
			</html>`))
		if err != nil {
			panic(err)
		}
	})

	log.Print("Listening on ", *listenAddress)
	log.Fatal(http.ListenAndServe(*listenAddress, nil))
}
