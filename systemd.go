// +build !nosystemd

package main

import (
	"flag"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/coreos/go-systemd/sdjournal"
)

// Journal represents a lockable systemd journal.
type Journal struct {
	*sdjournal.Journal
	sync.Mutex
	Path string
}

// NewJournal returns a Journal for reading journal entries.
func NewJournal(unit, slice, path string) (j *Journal, err error) {
	j = new(Journal)
	if path != "" {
		j.Journal, err = sdjournal.NewJournalFromDir(path)
		j.Path = path
	} else {
		j.Journal, err = sdjournal.NewJournal()
		j.Path = "journald"
	}
	if err != nil {
		return
	}

	if slice != "" {
		err = j.AddMatch("_SYSTEMD_SLICE=" + slice)
		if err != nil {
			return
		}
	} else if unit != "" {
		err = j.AddMatch("_SYSTEMD_UNIT=" + unit)
		if err != nil {
			return
		}
	}

	// Start at end of journal
	j.SeekRealtimeUsec(uint64(time.Now().UnixNano() / 1000))

	return
}

// NextMessage reads the next message from the journal.
func (j *Journal) NextMessage() (s string, c uint64, err error) {
	var e *sdjournal.JournalEntry

	// Read to next
	c, err = j.Next()
	if err != nil {
		return
	}
	// Return when on the end of journal
	if c == 0 {
		return
	}

	// Get entry
	e, err = j.GetEntry()
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
func systemdFlags(enable *bool, unit, slice, path *string) {
	flag.BoolVar(enable, "systemd.enable", false, "Read from the systemd journal instead of log")
	flag.StringVar(unit, "systemd.unit", "postfix.service", "Name of the Postfix systemd unit.")
	flag.StringVar(slice, "systemd.slice", "", "Name of the Postfix systemd slice. Overrides the systemd unit.")
	flag.StringVar(path, "systemd.journal_path", "", "Path to the systemd journal")
}

// CollectLogfileFromJournal Collects entries from the systemd journal.
func (e *PostfixExporter) CollectLogfileFromJournal() error {
	e.journal.Lock()
	defer e.journal.Unlock()

	r := e.journal.Wait(time.Duration(1) * time.Second)
	if r < 0 {
		log.Print("error while waiting for journal!")
	}
	for {
		m, c, err := e.journal.NextMessage()
		if err != nil {
			return err
		}
		if c == 0 {
			break
		}
		e.CollectFromLogline(m)
	}

	return nil
}
