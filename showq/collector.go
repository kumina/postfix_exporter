package showq

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"io"
	"log"
	"net"
	"regexp"
	"strconv"
	"time"
)

type GaugeVec interface {
	prometheus.Collector
	WithLabelValues(lvs ...string) prometheus.Gauge
}
type ShowQ struct {
	path           string
	sizeHistogram  prometheus.ObserverVec
	ageHistogram   prometheus.ObserverVec
	upGauge        GaugeVec
	scrapeInterval string
}

func NewShowQCollector(path string, upGauge GaugeVec, interval string) *ShowQ {
	return &ShowQ{
		path:           path,
		upGauge:        upGauge,
		scrapeInterval: interval,
		sizeHistogram: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "postfix",
				Name:      "showq_message_size_bytes",
				Help:      "Size of messages in Postfix's message queue, in bytes",
				Buckets:   []float64{1e3, 1e4, 1e5, 1e6, 1e7, 1e8, 1e9},
			},
			[]string{"queue"}),
		ageHistogram: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "postfix",
				Name:      "showq_message_age_seconds",
				Help:      "Age of messages in Postfix's message queue, in seconds",
				Buckets:   []float64{1e1, 1e2, 1e3, 1e4, 1e5, 1e6, 1e7, 1e8},
			},
			[]string{"queue"}),
	}
}

func (s *ShowQ) StartMetricCollection(ctx context.Context) (<-chan interface{}, error) {
	done := make(chan interface{})
	duration, err := time.ParseDuration(s.scrapeInterval)
	if err != nil {
		return nil, fmt.Errorf("failed to parse duration '%s': %v", s.scrapeInterval, err)
	}
	go func() {
		defer close(done)
		gauge := s.upGauge.WithLabelValues(s.path)
		ticker := time.NewTicker(duration)
		for {
			select {
			case <-ctx.Done():
				gauge.Set(0)
				return
			case <-ticker.C:
				err := s.CollectShowqFromSocket(s.path)
				if err == nil {
					gauge.Set(1)
				} else {
					log.Printf("Failed to scrape showq socket: %s", err)
					gauge.Set(0)
				}
			}
		}

	}()
	return done, nil
}

func (s ShowQ) Describe(ch chan<- *prometheus.Desc) {
	s.ageHistogram.Describe(ch)
	s.sizeHistogram.Describe(ch)
}

func (s ShowQ) Collect(ch chan<- prometheus.Metric) {
	s.sizeHistogram.Collect(ch)
	s.ageHistogram.Collect(ch)
}

// collectShowqFromSocket collects Postfix queue statistics from a socket.
func (s ShowQ) CollectShowqFromSocket(path string) error {
	fd, err := net.Dial("unix", path)
	if err != nil {
		return err
	}
	defer fd.Close()
	return s.collectShowqFromReader(fd)
}

// collectShowqFromReader parses the output of Postfix's 'showq' command
// and turns it into metrics.
//
// The output format of this command depends on the version of Postfix
// used. Postfix 2.x uses a textual format, identical to the output of
// the 'mailq' command. Postfix 3.x uses a binary format, where entries
// are terminated using null bytes. Auto-detect the format by scanning
// for null bytes in the first 128 bytes of output.
func (s ShowQ) collectShowqFromReader(file io.Reader) error {
	reader := bufio.NewReader(file)
	buf, err := reader.Peek(128)
	if err != nil && err != io.EOF {
		log.Printf("Could not read postfix output, %v", err)
	}
	if bytes.IndexByte(buf, 0) >= 0 {
		return s.collectBinaryShowqFromReader(reader)
	}
	return s.CollectTextualShowqFromScanner(reader)
}

func (s ShowQ) CollectTextualShowqFromScanner(file io.Reader) error {
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)
	// Initialize all queue buckets to zero.
	for _, q := range []string{"active", "hold", "other"} {
		s.sizeHistogram.WithLabelValues(q)
		s.ageHistogram.WithLabelValues(q)
	}

	location, err := time.LoadLocation("Local")
	if err != nil {
		log.Println(err)
	}

	// Regular expression for matching postqueue's output. Example:
	// "A07A81514      5156 Tue Feb 14 13:13:54  MAILER-DAEMON"
	messageLine := regexp.MustCompile(`^[0-9A-F]+([\*!]?) +(\d+) (\w{3} \w{3} +\d+ +\d+:\d{2}:\d{2}) +`)

	for scanner.Scan() {
		text := scanner.Text()
		matches := messageLine.FindStringSubmatch(text)
		if matches == nil {
			continue
		}
		queueMatch := matches[1]
		sizeMatch := matches[2]
		dateMatch := matches[3]

		// Derive the name of the message queue.
		queue := "other"
		if queueMatch == "*" {
			queue = "active"
		} else if queueMatch == "!" {
			queue = "hold"
		}

		// Parse the message size.
		size, err := strconv.ParseFloat(sizeMatch, 64)
		if err != nil {
			return err
		}

		// Parse the message date. Unfortunately, the
		// output contains no year number. Assume it
		// applies to the last year for which the
		// message date doesn't exceed time.Now().
		date, err := time.ParseInLocation("Mon Jan 2 15:04:05", dateMatch, location)
		if err != nil {
			return err
		}
		now := time.Now()
		date = date.AddDate(now.Year(), 0, 0)
		if date.After(now) {
			date = date.AddDate(-1, 0, 0)
		}

		s.sizeHistogram.WithLabelValues(queue).Observe(size)
		s.ageHistogram.WithLabelValues(queue).Observe(now.Sub(date).Seconds())
	}
	return scanner.Err()
}

// scanNullTerminatedEntries is a splitting function for bufio.Scanner
// to split entries by null bytes.
func (s ShowQ) scanNullTerminatedEntries(data []byte, atEOF bool) (advance int, token []byte, err error) {
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

// collectBinaryShowqFromReader parses Postfix's binary showq format.
func (s ShowQ) collectBinaryShowqFromReader(file io.Reader) error {
	scanner := bufio.NewScanner(file)
	scanner.Split(s.scanNullTerminatedEntries)

	// Histograms tracking the messages by size and age.
	s.sizeHistogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "postfix",
			Name:      "showq_message_size_bytes",
			Help:      "Size of messages in Postfix's message queue, in bytes",
			Buckets:   []float64{1e3, 1e4, 1e5, 1e6, 1e7, 1e8, 1e9},
		},
		[]string{"queue"})
	s.ageHistogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "postfix",
			Name:      "showq_message_age_seconds",
			Help:      "Age of messages in Postfix's message queue, in seconds",
			Buckets:   []float64{1e1, 1e2, 1e3, 1e4, 1e5, 1e6, 1e7, 1e8},
		},
		[]string{"queue"})

	// Initialize all queue buckets to zero.
	for _, q := range []string{"active", "deferred", "hold", "incoming", "maildrop"} {
		s.sizeHistogram.WithLabelValues(q)
		s.ageHistogram.WithLabelValues(q)
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
			s.sizeHistogram.WithLabelValues(queue).Observe(size)
		} else if key == "time" {
			// Message time as a UNIX timestamp.
			utime, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return err
			}
			s.ageHistogram.WithLabelValues(queue).Observe(now - utime)
		}
	}

	return scanner.Err()
}
