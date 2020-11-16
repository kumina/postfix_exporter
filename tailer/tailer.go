package tailer

import (
	"context"
	"fmt"
	"io"

	"github.com/nxadm/tail"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

func TailLog(ctx context.Context, filename string) (<-chan string, error) {
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
