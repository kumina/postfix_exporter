// +build !nosystemd,linux

package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/alecthomas/kingpin"
	"github.com/coreos/go-systemd/v22/sdjournal"
)

// timeNow is a test fake injection point.
var timeNow = time.Now

// A SystemdLogSource reads log records from the given Systemd
// journal.
type SystemdLogSource struct {
	journal SystemdJournal
	path    string
}

// A SystemdJournal is the journal interface that sdjournal.Journal
// provides. See https://pkg.go.dev/github.com/coreos/go-systemd/sdjournal?tab=doc
type SystemdJournal interface {
	io.Closer
	AddMatch(match string) error
	GetEntry() (*sdjournal.JournalEntry, error)
	Next() (uint64, error)
	SeekRealtimeUsec(usec uint64) error
	Wait(timeout time.Duration) int
}

// NewSystemdLogSource returns a log source for reading Systemd
// journal entries. `unit` and `slice` provide filtering if non-empty
// (with `slice` taking precedence).
func NewSystemdLogSource(j SystemdJournal, path, unit, slice string) (*SystemdLogSource, error) {
	logSrc := &SystemdLogSource{journal: j, path: path}

	var err error
	if slice != "" {
		err = logSrc.journal.AddMatch("_SYSTEMD_SLICE=" + slice)
	} else if unit != "" {
		err = logSrc.journal.AddMatch("_SYSTEMD_UNIT=" + unit)
	}
	if err != nil {
		logSrc.journal.Close()
		return nil, err
	}

	// Start at end of journal
	if err := logSrc.journal.SeekRealtimeUsec(uint64(timeNow().UnixNano() / 1000)); err != nil {
		logSrc.journal.Close()
		return nil, err
	}

	if r := logSrc.journal.Wait(1 * time.Second); r < 0 {
		logSrc.journal.Close()
		return nil, err
	}

	return logSrc, nil
}

func (s *SystemdLogSource) Close() error {
	return s.journal.Close()
}

func (s *SystemdLogSource) Path() string {
	return s.path
}

func (s *SystemdLogSource) Read(ctx context.Context) (string, error) {
	c, err := s.journal.Next()
	if err != nil {
		return "", err
	}
	if c == 0 {
		return "", io.EOF
	}

	e, err := s.journal.GetEntry()
	if err != nil {
		return "", err
	}
	ts := time.Unix(0, int64(e.RealtimeTimestamp)*int64(time.Microsecond))

	return fmt.Sprintf(
		"%s %s %s[%s]: %s",
		ts.Format(time.Stamp),
		e.Fields["_HOSTNAME"],
		e.Fields["SYSLOG_IDENTIFIER"],
		e.Fields["_PID"],
		e.Fields["MESSAGE"],
	), nil
}

// A systemdLogSourceFactory is a factory that can create
// SystemdLogSources from command line flags.
type systemdLogSourceFactory struct {
	enable            bool
	unit, slice, path string
}

func (f *systemdLogSourceFactory) Init(app *kingpin.Application) {
	app.Flag("systemd.enable", "Read from the systemd journal instead of log").Default("false").BoolVar(&f.enable)
	app.Flag("systemd.unit", "Name of the Postfix systemd unit.").Default("postfix.service").StringVar(&f.unit)
	app.Flag("systemd.slice", "Name of the Postfix systemd slice. Overrides the systemd unit.").Default("").StringVar(&f.slice)
	app.Flag("systemd.journal_path", "Path to the systemd journal").Default("").StringVar(&f.path)
}

func (f *systemdLogSourceFactory) New() (LogSourceCloser, error) {
	if !f.enable {
		return nil, nil
	}

	log.Println("Reading log events from systemd")
	j, path, err := newSystemdJournal(f.path)
	if err != nil {
		return nil, err
	}
	return NewSystemdLogSource(j, path, f.unit, f.slice)
}

// newSystemdJournal creates a journal handle. It returns the handle
// and a string representation of it. If `path` is empty, it connects
// to the local journald.
func newSystemdJournal(path string) (*sdjournal.Journal, string, error) {
	if path != "" {
		j, err := sdjournal.NewJournalFromDir(path)
		return j, path, err
	}

	j, err := sdjournal.NewJournal()
	return j, "journald", err
}

func init() {
	RegisterLogSourceFactory(&systemdLogSourceFactory{})
}
