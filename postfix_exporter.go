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
	"flag"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	postfixUpDesc = prometheus.NewDesc(
		prometheus.BuildFQName("unbound", "", "up"),
		"Whether scraping Postfix's metrics was successful.",
		nil, nil)
)

func CollectShowqFromReader(file io.Reader, ch chan<- prometheus.Metric) error {
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)

	// Regular expression for matching postqueue's output. Example:
	// "A07A81514      5156 Tue Feb 14 13:13:54  MAILER-DAEMON"
	messageLine := regexp.MustCompile("^[0-9A-F]+ +(\\d+) (\\w{3} \\w{3} +\\d+ +\\d+:\\d{2}:\\d{2}) +")

	// Histograms tracking the messages by size and age.
	sizeHistogram := prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "unbound",
			Name:      "queue_message_size_bytes",
			Help:      "Size of messages in Postfix's message queue, in bytes",
			Buckets:   []float64{1e3, 1e4, 1e5, 1e6, 1e7, 1e8, 1e9},
		})
	ageHistogram := prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "unbound",
			Name:      "queue_message_age_seconds",
			Help:      "Age of messages in Postfix's message queue, in seconds",
			Buckets:   []float64{1e1, 1e2, 1e3, 1e4, 1e5, 1e6, 1e7, 1e8},
		})

	now := time.Now()
	for scanner.Scan() {
		matches := messageLine.FindStringSubmatch(scanner.Text())
		if matches != nil {
			// Parse the message size.
			size, err := strconv.ParseFloat(matches[1], 64)
			if err != nil {
				return err
			}

			// Parse the message date. Unfortunately, the
			// output contains no year number. Assume it
			// applies to the last year for which the
			// message date doesn't exceed time.Now().
			date, err := time.Parse("Mon Jan 2 15:04:05", matches[2])
			if err != nil {
				return err
			}
			date = date.AddDate(now.Year(), 0, 0)
			if date.After(now) {
				date.AddDate(-1, 0, 0)
			}

			sizeHistogram.Observe(size)
			ageHistogram.Observe(now.Sub(date).Seconds())
		}
	}

	ch <- sizeHistogram
	ch <- ageHistogram
	return scanner.Err()
}

func CollectShowqFromFile(path string, ch chan<- prometheus.Metric) error {
	conn, err := os.Open(path)
	if err != nil {
		return err
	}
	return CollectShowqFromReader(conn, ch)
}

func CollectShowqFromSocket(path string, ch chan<- prometheus.Metric) error {
	conn, err := net.Dial("unix", path)
	if err != nil {
		return err
	}
	return CollectShowqFromReader(conn, ch)
}

type PostfixExporter struct {
	showqPath string
}

func NewPostfixExporter(showqPath string) (*PostfixExporter, error) {
	return &PostfixExporter{
		showqPath: showqPath,
	}, nil
}

func (e *PostfixExporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- postfixUpDesc
}

func (e *PostfixExporter) Collect(ch chan<- prometheus.Metric) {
	err := CollectShowqFromSocket(e.showqPath, ch)
	if err == nil {
		ch <- prometheus.MustNewConstMetric(
			postfixUpDesc,
			prometheus.GaugeValue,
			1.0)
	} else {
		log.Printf("Failed to scrape showq socket: %s", err)
		ch <- prometheus.MustNewConstMetric(
			postfixUpDesc,
			prometheus.GaugeValue,
			0.0)
	}
}

func main() {
	var (
		listenAddress    = flag.String("web.listen-address", ":12345", "Address to listen on for web interface and telemetry.")
		metricsPath      = flag.String("web.telemetry-path", "/metrics", "Path under which to expose metrics.")
		postfixShowqPath = flag.String("postfix.showq_path", "/var/spool/postfix/public/showq", "Path at which Postfix places its showq socket.")
	)
	flag.Parse()

	exporter, err := NewPostfixExporter(*postfixShowqPath)
	if err != nil {
		panic(err)
	}
	prometheus.MustRegister(exporter)

	http.Handle(*metricsPath, prometheus.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`
			<html>
			<head><title>Postfix Exporter</title></head>
			<body>
			<h1>Postfix Exporter</h1>
			<p><a href='` + *metricsPath + `'>Metrics</a></p>
			</body>
			</html>`))
	})
	log.Fatal(http.ListenAndServe(*listenAddress, nil))
}
