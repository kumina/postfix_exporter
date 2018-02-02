package main

import (
	"fmt"
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
