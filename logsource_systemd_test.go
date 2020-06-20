// +build !nosystemd,linux

package main

import (
	"context"
	"io"
	"os"
	"testing"
	"time"

	"github.com/coreos/go-systemd/v22/sdjournal"
	"github.com/stretchr/testify/assert"
)

func TestNewSystemdLogSource(t *testing.T) {
	j := &fakeSystemdJournal{}
	src, err := NewSystemdLogSource(j, "apath", "aunit", "aslice")
	if err != nil {
		t.Fatalf("NewSystemdLogSource failed: %v", err)
	}

	assert.Equal(t, []string{"_SYSTEMD_SLICE=aslice"}, j.addMatchCalls, "A match should be added for slice.")
	assert.Equal(t, []uint64{1234567890000000}, j.seekRealtimeUsecCalls, "A call to SeekRealtimeUsec should be made.")
	assert.Equal(t, []time.Duration{1 * time.Second}, j.waitCalls, "A call to Wait should be made.")

	if err := src.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	assert.Equal(t, 1, j.closeCalls, "A call to Close should be made.")
}

func TestSystemdLogSource_Path(t *testing.T) {
	j := &fakeSystemdJournal{}
	src, err := NewSystemdLogSource(j, "apath", "aunit", "aslice")
	if err != nil {
		t.Fatalf("NewSystemdLogSource failed: %v", err)
	}
	defer src.Close()

	assert.Equal(t, "apath", src.Path(), "Path should be set by New.")
}

func TestSystemdLogSource_Read(t *testing.T) {
	ctx := context.Background()

	j := &fakeSystemdJournal{
		getEntryValues: []sdjournal.JournalEntry{
			{
				Fields: map[string]string{
					"_HOSTNAME":         "ahost",
					"SYSLOG_IDENTIFIER": "anid",
					"_PID":              "123",
					"MESSAGE":           "aline",
				},
				RealtimeTimestamp: 1234567890000000,
			},
		},
		nextValues: []uint64{1},
	}
	src, err := NewSystemdLogSource(j, "apath", "aunit", "aslice")
	if err != nil {
		t.Fatalf("NewSystemdLogSource failed: %v", err)
	}
	defer src.Close()

	s, err := src.Read(ctx)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	assert.Equal(t, "Feb 13 23:31:30 ahost anid[123]: aline", s, "Read should get data from the journal entry.")
}

func TestSystemdLogSource_ReadEOF(t *testing.T) {
	ctx := context.Background()

	j := &fakeSystemdJournal{
		nextValues: []uint64{0},
	}
	src, err := NewSystemdLogSource(j, "apath", "aunit", "aslice")
	if err != nil {
		t.Fatalf("NewSystemdLogSource failed: %v", err)
	}
	defer src.Close()

	_, err = src.Read(ctx)
	assert.Equal(t, io.EOF, err, "Should interpret Next 0 as EOF.")
}

func TestMain(m *testing.M) {
	// We compare Unix timestamps to date strings, so make it deterministic.
	os.Setenv("TZ", "UTC")
	timeNow = func() time.Time { return time.Date(2009, 2, 13, 23, 31, 30, 0, time.UTC) }
	defer func() {
		timeNow = time.Now
	}()

	os.Exit(m.Run())
}

type fakeSystemdJournal struct {
	getEntryValues []sdjournal.JournalEntry
	getEntryError  error
	nextValues     []uint64
	nextError      error

	addMatchCalls         []string
	closeCalls            int
	seekRealtimeUsecCalls []uint64
	waitCalls             []time.Duration
}

func (j *fakeSystemdJournal) AddMatch(match string) error {
	j.addMatchCalls = append(j.addMatchCalls, match)
	return nil
}

func (j *fakeSystemdJournal) Close() error {
	j.closeCalls++
	return nil
}

func (j *fakeSystemdJournal) GetEntry() (*sdjournal.JournalEntry, error) {
	if len(j.getEntryValues) == 0 {
		return nil, j.getEntryError
	}
	e := j.getEntryValues[0]
	j.getEntryValues = j.getEntryValues[1:]
	return &e, nil
}

func (j *fakeSystemdJournal) Next() (uint64, error) {
	if len(j.nextValues) == 0 {
		return 0, j.nextError
	}
	v := j.nextValues[0]
	j.nextValues = j.nextValues[1:]
	return v, nil
}

func (j *fakeSystemdJournal) SeekRealtimeUsec(usec uint64) error {
	j.seekRealtimeUsecCalls = append(j.seekRealtimeUsecCalls, usec)
	return nil
}

func (j *fakeSystemdJournal) Wait(timeout time.Duration) int {
	j.waitCalls = append(j.waitCalls, timeout)
	return 0
}
