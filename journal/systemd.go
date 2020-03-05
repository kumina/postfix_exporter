// +build !nosystemd,linux

package journal

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/alecthomas/kingpin"
	"github.com/coreos/go-systemd/v22/sdjournal"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

// Journal represents a lockable systemd journal.
type Journal struct {
	journal             *sdjournal.Journal
	mu                  sync.Mutex
	Path                string
	linesCollectedGauge prometheus.Counter
}

func (j *Journal) Describe(ch chan<- *prometheus.Desc) {
	j.linesCollectedGauge.Describe(ch)
}

func (j *Journal) Collect(ch chan<- prometheus.Metric) {
	j.linesCollectedGauge.Collect(ch)
}

// NewJournal returns a Journal for reading journal entries.
func NewJournal(unit, slice, path string) (*Journal, error) {
	j := &Journal{
		linesCollectedGauge: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace:   "postfix_exporter",
			Subsystem:   "",
			Name:        "lines_collected",
			Help:        "Number of lines collected by PostfixExporter",
			ConstLabels: prometheus.Labels{"source": "journald"},
		}),
	}
	var err error
	if path != "" {
		j.journal, err = sdjournal.NewJournalFromDir(path)
		j.Path = path
	} else {
		j.journal, err = sdjournal.NewJournal()
		j.Path = "journald"
	}
	if err != nil {
		return nil, fmt.Errorf("failed to open journal: %v", err)
	}

	if slice != "" {
		err = j.journal.AddMatch("_SYSTEMD_SLICE=" + slice)
		if err != nil {
			return nil, fmt.Errorf("failed to add match for slice: %v", err)
		}
	} else if unit != "" {
		err = j.journal.AddMatch("_SYSTEMD_UNIT=" + unit)
		if err != nil {
			return nil, fmt.Errorf("failed to add match for unit: %v", err)
		}
	}

	// Start at end of journal
	err = j.journal.SeekRealtimeUsec(uint64(time.Now().UnixNano() / 1000))
	if err != nil {
		return nil, fmt.Errorf("failed to seek to end: %v", err)
	}
	return j, nil
}

// NextMessage reads the next message from the journal.
func (j *Journal) NextMessage() (s string, c uint64, err error) {
	var e *sdjournal.JournalEntry

	// Read to next
	c, err = j.journal.Next()
	if err != nil {
		return
	}
	// Return when on the end of journal
	if c == 0 {
		return
	}

	// Get entry
	e, err = j.journal.GetEntry()
	if err != nil {
		return
	}
	ts := time.Unix(0, int64(e.RealtimeTimestamp)*int64(time.Microsecond))

	// Format entry
	s = fmt.Sprintf(
		"%s %s %s[%s]: %s",
		ts.Format(time.Stamp),
		e.Fields["_HOSTNAME"],
		e.Fields["SYSLOG_IDENTIFIER"],
		e.Fields["_PID"],
		e.Fields["MESSAGE"],
	)

	return
}

// systemdFlags sets the flags for use with systemd
func SystemdFlags(enable *bool, unit, slice, path *string, app *kingpin.Application) {
	app.Flag("systemd.enable", "Read from the systemd journal instead of log").Default("false").BoolVar(enable)
	app.Flag("systemd.unit", "Name of the Postfix systemd unit.").Default("postfix.service").StringVar(unit)
	app.Flag("systemd.slice", "Name of the Postfix systemd slice. Overrides the systemd unit.").Default("").StringVar(slice)
	app.Flag("systemd.journal_path", "Path to the systemd journal").Default("").StringVar(path)
}

// CollectLogfileFromJournal Collects entries from the systemd journal.
func (j *Journal) CollectLogLinesFromJournal(ctx context.Context) (<-chan string, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	lines := make(chan string)

	r := j.journal.Wait(time.Duration(1) * time.Second)
	if r < 0 {
		log.Print("error while waiting for journal!")
	}
	go func() {
		logrus.Info("Started journal tailing.")
		defer close(lines)
		for {
			select {
			case <-ctx.Done():
				logrus.Printf("Leaving systemd collection")
				return
			default:
				m, c, err := j.NextMessage()
				if err != nil {
					logrus.Printf("failed to get next message: %v", err)
					return
				}
				if c == 0 {
					j.journal.Wait(time.Duration(1) * time.Second)
					continue
				} else {
					j.linesCollectedGauge.Inc()
					lines <- m
				}
			}

		}
	}()

	return lines, nil
}

func (j *Journal) Close() {
	j.journal.Close()
}
