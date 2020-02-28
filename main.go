package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/alecthomas/kingpin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	var (
		app                                           = kingpin.New("postfix_exporter", "Prometheus metrics exporter for postfix")
		listenAddress                                 = app.Flag("web.listen-address", "Address to listen on for web interface and telemetry.").Default(":9154").String()
		metricsPath                                   = app.Flag("web.telemetry-path", "Path under which to expose metrics.").Default("/metrics").String()
		postfixShowqPath                              = app.Flag("postfix.showq_path", "Path at which Postfix places its showq socket.").Default("/var/spool/postfix/public/showq").String()
		postfixLogfilePath                            = app.Flag("postfix.logfile_path", "Path where Postfix writes log entries.").Default("/var/log/maillog").String()
		logUnsupportedLines                           = app.Flag("log.unsupported", "Log all unsupported lines.").Bool()
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
		log.Println("Reading log events from systemd")
	} else {
		log.Printf("Reading log events from %v", *postfixLogfilePath)
	}

	exporter, err := NewPostfixExporter(
		*postfixShowqPath,
		*postfixLogfilePath,
		journal,
		*logUnsupportedLines,
	)
	if err != nil {
		log.Fatalf("Failed to create PostfixExporter: %s", err)
	}
	prometheus.MustRegister(exporter)

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
	collectionFinished := exporter.StartMetricCollection(ctx)
	log.Print("Listening on ", *listenAddress)
	var srv http.Server = http.Server{Addr: *listenAddress}
	idleConnsClosed := catchInterruptAndShutdownServer(ctx, srv)

	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		// Error starting or closing listener:
		log.Fatalf("HTTP server ListenAndServe: %v", err)
	}
	log.Print("Cleanup")
	cancelFunc()
	log.Print("Canceled")
	<-idleConnsClosed
	log.Print("Idles closed")
	<-collectionFinished
	log.Print("Shutdown completed")
}

func catchInterruptAndShutdownServer(ctx context.Context, srv http.Server) <-chan struct{} {
	idleConnsClosed := make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt)
		<-sigint
		// Give Server 1 second to cleanly shut down
		d := time.Now().Add(100 * time.Millisecond)
		// We received an interrupt signal, shut down.
		deadlineCtx, _ := context.WithDeadline(ctx, d)
		log.Printf("Shutting down. Waiting for %v", d)
		if err := srv.Shutdown(deadlineCtx); err != nil {
			// Error from closing listeners, or context timeout:
			log.Printf("HTTP server Shutdown: %v", err)
		}
		log.Print("Shutdown completed")
		close(idleConnsClosed)
	}()
	return idleConnsClosed
}
