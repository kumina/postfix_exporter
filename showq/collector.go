package showq

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
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
	mu             sync.RWMutex
}

const queueOther = "other"
const queueHold = "hold"
const queueActive = "active"
const queueIncoming = "incoming"

func NewShowQCollector(path string, upGauge GaugeVec, interval string) *ShowQ {
	histograms := NewHistograms()
	return &ShowQ{
		path:           path,
		upGauge:        upGauge,
		scrapeInterval: interval,
		sizeHistogram:  histograms.SizeHistogram,
		ageHistogram:   histograms.AgeHistogram,
	}
}

func (s *ShowQ) StartMetricCollection(ctx context.Context, location string) (<-chan interface{}, error) {
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
				s.collectMetrics(gauge, location)
			}
		}

	}()
	return done, nil
}

func (s *ShowQ) collectMetrics(gauge prometheus.Gauge, location string) {
	hist := NewHistograms()
	err := CollectShowqFromSocket(s.path, hist, location)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sizeHistogram = hist.SizeHistogram
	s.ageHistogram = hist.AgeHistogram
	if err == nil {
		gauge.Set(1)
	} else {
		log.Printf("Failed to scrape showq socket: %s", err)
		gauge.Set(0)
	}
}

func (s *ShowQ) Describe(ch chan<- *prometheus.Desc) {
	s.ageHistogram.Describe(ch)
	s.sizeHistogram.Describe(ch)
}

func (s *ShowQ) Collect(ch chan<- prometheus.Metric) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	s.sizeHistogram.Collect(ch)
	s.ageHistogram.Collect(ch)
}

// collectShowqFromSocket collects Postfix queue statistics from a socket.
func CollectShowqFromSocket(path string, hist Histograms, location string) error {
	fd, err := net.Dial("unix", path)
	if err != nil {
		return err
	}
	defer fd.Close()
	return collectShowqFromReader(fd, hist, location)
}

// collectShowqFromReader parses the output of Postfix's 'showq' command
// and turns it into metrics.
//
// The output format of this command depends on the version of Postfix
// used. Postfix 2.x uses a textual format, identical to the output of
// the 'mailq' command. Postfix 3.x uses a binary format, where entries
// are terminated using null bytes. Auto-detect the format by scanning
// for null bytes in the first 128 bytes of output.
func collectShowqFromReader(file io.Reader, hist Histograms, location string) error {
	reader := bufio.NewReader(file)
	buf, err := reader.Peek(128)
	if err != nil && err != io.EOF {
		log.Printf("Could not read postfix output, %v", err)
	}
	if bytes.IndexByte(buf, 0) >= 0 {
		return CollectBinaryShowqFromReader(reader, hist)
	}
	return CollectTextualShowqFromScanner(reader, hist, location)
}

func CollectTextualShowqFromScanner(file io.Reader, hist Histograms, locationName string) error {
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)
	// Initialize all queue buckets to zero.

	location, err := time.LoadLocation(locationName)
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
		queue := queueOther
		if queueMatch == "*" {
			queue = queueActive
		} else if queueMatch == "!" {
			queue = queueHold
		}

		// Parse the message size.
		size, err := strconv.ParseFloat(sizeMatch, 64)
		if err != nil {
			return err
		}

		seconds, err := ParseDateToSeconds(dateMatch, location)
		if err != nil {
			return err
		}

		hist.SizeHistogram.WithLabelValues(queue).Observe(size)
		hist.AgeHistogram.WithLabelValues(queue).Observe(seconds)
	}
	return scanner.Err()
}

func ParseDateToSeconds(dateMatch string, location *time.Location) (float64, error) {
	// Parse the message date. Unfortunately, the
	// output contains no year number. Assume it
	// applies to the last year for which the
	// message date doesn't exceed time.Now().
	date, err := time.ParseInLocation("Mon Jan 2 15:04:05", dateMatch, location)
	if err != nil {
		return 0, err
	}
	now := time.Now()
	date = date.AddDate(now.Year(), 0, 0)
	if date.After(now) {
		date = date.AddDate(-1, 0, 0)
	}
	seconds := now.Sub(date).Seconds()
	return seconds, nil
}

// scanNullTerminatedEntries is a splitting function for bufio.Scanner
// to split entries by null bytes.
func scanNullTerminatedEntries(data []byte, atEOF bool) (advance int, token []byte, err error) {
	i := bytes.IndexByte(data, 0)
	switch {
	case i >= 0:
		// Valid record found.
		return i + 1, data[0:i], nil
	case atEOF && len(data) != 0:
		// Data at the end of the file without a null terminator.
		return 0, nil, errors.New("Expected null byte terminator")
	default:
		// Request more data.
		return 0, nil, nil
	}
}

// collectBinaryShowqFromReader parses Postfix's binary showq format.
func CollectBinaryShowqFromReader(file io.Reader, hist Histograms) error {
	scanner := bufio.NewScanner(file)
	scanner.Split(scanNullTerminatedEntries)

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
		switch key {
		case "queue_name":
			// The name of the message queue.
			queue = value
		case "size":
			// Message size in bytes.
			size, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return err
			}
			hist.SizeHistogram.WithLabelValues(queue).Observe(size)
		case "time":
			// Message time as a UNIX timestamp.
			utime, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return err
			}
			hist.AgeHistogram.WithLabelValues(queue).Observe(now - utime)
		}
	}

	return scanner.Err()
}
