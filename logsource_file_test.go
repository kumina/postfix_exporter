package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFileLogSource_Path(t *testing.T) {
	path, close, err := setupFakeLogFile()
	if err != nil {
		t.Fatalf("setupFakeTailer failed: %v", err)
	}
	defer close()

	src, err := NewFileLogSource(path)
	if err != nil {
		t.Fatalf("NewFileLogSource failed: %v", err)
	}
	defer src.Close()

	assert.Equal(t, path, src.Path(), "Path should be set by New.")
}

func TestFileLogSource_Read(t *testing.T) {
	ctx := context.Background()

	path, close, err := setupFakeLogFile()
	if err != nil {
		t.Fatalf("setupFakeTailer failed: %v", err)
	}
	defer close()

	src, err := NewFileLogSource(path)
	if err != nil {
		t.Fatalf("NewFileLogSource failed: %v", err)
	}
	defer src.Close()

	s, err := src.Read(ctx)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	assert.Equal(t, "Feb 13 23:31:30 ahost anid[123]: aline", s, "Read should get data from the journal entry.")
}

func setupFakeLogFile() (string, func(), error) {
	f, err := ioutil.TempFile("", "filelogsource")
	if err != nil {
		return "", nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer os.Remove(f.Name())
		defer f.Close()

		for {
			// The tailer seeks to the end and then does a
			// follow. Keep writing lines so we know it wakes up and
			// returns lines.
			fmt.Fprintln(f, "Feb 13 23:31:30 ahost anid[123]: aline")

			select {
			case <-time.After(10 * time.Millisecond):
				// continue
			case <-ctx.Done():
				return
			}
		}
	}()

	return f.Name(), func() {
		cancel()
		wg.Wait()
	}, nil
}
