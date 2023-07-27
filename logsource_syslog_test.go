package main

import (
	"context"
	"fmt"
	"log/syslog"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSyslogLogSource(t *testing.T) {
	src, err := NewSyslogLogSource("udp", "localhost:8514")
	if err != nil {
		t.Fatalf("NewSyslogLogSource failed: %v", err)
	}

	if err := src.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}

func TestSyslogLogSource_Path(t *testing.T) {
	src, err := NewSyslogLogSource("udp", "localhost:8514")
	if err != nil {
		t.Fatalf("NewSyslogLogSource failed: %v", err)
	}
	defer src.Close()

	assert.Equal(t, "udp:localhost:8514", src.Path(), "Path should be set by New.")
}

func TestSyslogLogSource_Read(t *testing.T) {
	ctx := context.Background()

	src, err := NewSyslogLogSource("udp", "localhost:8514")
	if err != nil {
		t.Fatalf("NewSyslogLogSource failed: %v", err)
	}
	defer src.Close()

	sysLog, err := syslog.Dial("udp", "localhost:8514", syslog.LOG_MAIL|syslog.LOG_INFO, "tag")
	fmt.Fprint(sysLog, "a line")

	s, err := src.Read(ctx)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	assert.Contains(t, s, "a line")
}
