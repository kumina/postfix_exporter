package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/alecthomas/kingpin"
	"github.com/hpcloud/tail"
	"github.com/kumina/postfix_exporter/logCollector"
	"github.com/kumina/postfix_exporter/showq"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

func main() {
	var (
		app                                           = kingpin.New("postfix_exporter", "Prometheus metrics logFileCollector for postfix")
		listenAddress                                 = app.Flag("web.listen-address", "Address to listen on for web interface and telemetry.").Default(":9154").String()
		metricsPath                                   = app.Flag("web.telemetry-path", "Path under which to expose metrics.").Default("/metrics").String()
		postfixShowqPath                              = app.Flag("postfix.showq_path", "Path at which Postfix places its showq socket.").Default("/var/spool/postfix/public/showq").String()
		postfixShowqInterval                          = app.Flag("postfix.showq_interval", "Interval between showq scrapes.").Default("10s").String()
		postfixLogfilePath                            = app.Flag("postfix.logfile_path", "Path where Postfix writes log entries.").Default("/var/log/maillog").String()
		logUnsupportedLines                           = app.Flag("log.unsupported", "Log all unsupported lines.").Bool()
		systemdEnable                                 bool
		systemdUnit, systemdSlice, systemdJournalPath string
	)
	logCollector.SystemdFlags(&systemdEnable, &systemdUnit, &systemdSlice, &systemdJournalPath, app)

	kingpin.MustParse(app.Parse(os.Args[1:]))

	var journal *logCollector.Journal
	if systemdEnable {
		var err error
		journal, err = logCollector.NewJournal(systemdUnit, systemdSlice, systemdJournalPath)
		if err != nil {
			log.Fatalf("Error opening systemd journal: %s", err)
		}
		defer journal.Close()
		prometheus.MustRegister(journal)
		log.Println("Reading log events from systemd")
	} else {
		log.Printf("Reading log events from %v", *postfixLogfilePath)
	}
	postfixUp := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "postfix",
			Subsystem: "",
			Name:      "up",
			Help:      "Whether scraping Postfix's metrics was successful.",
		},
		[]string{"path"})
	prometheus.MustRegister(postfixUp)

	showQ := showq.NewShowQCollector(*postfixShowqPath, postfixUp, *postfixShowqInterval)
	prometheus.MustRegister(showQ)

	logFileCollector, err := logCollector.NewLogCollector(*logUnsupportedLines, postfixUp)
	if err != nil {
		log.Fatalf("Failed to create LogCollector: %s", err)
	}
	prometheus.MustRegister(logFileCollector)

	http.Handle(*metricsPath, promhttp.Handler())
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
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	var logLines <-chan string

	if journal == nil {
		logLines, err = tailLog(ctx, *postfixLogfilePath)
		if err != nil {
			logrus.Errorf("Failed to start tailing the logfile %s: %v", *postfixLogfilePath, err)
		}
	} else {
		logLines, err = journal.CollectLogLinesFromJournal(ctx)
	}

	logCollectionDone := logFileCollector.StartMetricCollection(ctx, logLines)
	showqCollectionDone, err := showQ.StartMetricCollection(ctx)
	if err != nil {
		log.Printf("failed to start showq metrics collection: %v", err)
	}
	log.Print("Listening on ", *listenAddress)

	var srv = http.Server{
		Addr:         *listenAddress,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	go func() {
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			// Error starting or closing listener:
			log.Fatalf("HTTP server ListenAndServe: %v", err)
		}
	}()

	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt)
	<-sigint
	log.Print("Shutting down")
	timeoutCtx, c := context.WithTimeout(ctx, 15*time.Second)
	defer c()
	err = srv.Shutdown(timeoutCtx)
	if err != nil {
		log.Print(err)
	}
	cancelFunc()
	<-logCollectionDone
	<-showqCollectionDone
	logrus.Print("Shutdown completed")
	os.Exit(0)
}

type tailer struct {
}

func tailLog(ctx context.Context, filename string) (<-chan string, error) {
	tailer, err := tail.TailFile(filename, tail.Config{
		ReOpen:    true,                               // reopen the file if it's rotated
		MustExist: true,                               // fail immediately if the file is missing or has incorrect permissions
		Follow:    true,                               // run in follow mode
		Location:  &tail.SeekInfo{Whence: io.SeekEnd}, // seek to end of file
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start tailer: %v", err)
	}
	lines := make(chan string)
	go func() {
		counter := prometheus.NewCounter(prometheus.CounterOpts{
			Namespace:   "postfix_exporter",
			Subsystem:   "",
			Name:        "lines_collected",
			Help:        "Number of lines collected by PostfixExporter",
			ConstLabels: prometheus.Labels{"source": "tailer"},
		})
		prometheus.MustRegister(counter)
		for {
			select {
			case <-ctx.Done():
				err := tailer.Stop()
				logrus.Printf("failed to stop tailing: %v", err)
				close(lines)
				return
			case line, ok := <-tailer.Lines:
				if !ok {
					logrus.Printf("tailer seems to be finished.")
					close(lines)
					return
				}
				if line.Err != nil {
					logrus.Printf("failed to receive line: %v", line.Err)
				}
				counter.Inc()
				lines <- line.Text
			}
		}
	}()
	return lines, nil
}
