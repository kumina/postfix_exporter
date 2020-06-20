package main

import (
	"context"
	"io"
	"log"

	"github.com/alecthomas/kingpin"
	"github.com/hpcloud/tail"
)

// A FileLogSource can read lines from a file.
type FileLogSource struct {
	tailer *tail.Tail
}

// NewFileLogSource creates a new log source, tailing the given file.
func NewFileLogSource(path string) (*FileLogSource, error) {
	tailer, err := tail.TailFile(path, tail.Config{
		ReOpen:    true,                               // reopen the file if it's rotated
		MustExist: true,                               // fail immediately if the file is missing or has incorrect permissions
		Follow:    true,                               // run in follow mode
		Location:  &tail.SeekInfo{Whence: io.SeekEnd}, // seek to end of file
		Logger:    tail.DiscardingLogger,
	})
	if err != nil {
		return nil, err
	}
	return &FileLogSource{tailer}, nil
}

func (s *FileLogSource) Close() error {
	defer s.tailer.Cleanup()
	go func() {
		// Stop() waits for the tailer goroutine to shut down, but it
		// can be blocking on sending on the Lines channel...
		for range s.tailer.Lines {
		}
	}()
	return s.tailer.Stop()
}

func (s *FileLogSource) Path() string {
	return s.tailer.Filename
}

func (s *FileLogSource) Read(ctx context.Context) (string, error) {
	select {
	case line, ok := <-s.tailer.Lines:
		if !ok {
			return "", io.EOF
		}
		return line.Text, nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

// A fileLogSourceFactory is a factory than can create log sources
// from command line flags.
//
// Because this factory is enabled by default, it must always be
// registered last.
type fileLogSourceFactory struct {
	path string
}

func (f *fileLogSourceFactory) Init(app *kingpin.Application) {
	app.Flag("postfix.logfile_path", "Path where Postfix writes log entries.").Default("/var/log/maillog").StringVar(&f.path)
}

func (f *fileLogSourceFactory) New() (LogSourceCloser, error) {
	if f.path == "" {
		return nil, nil
	}
	log.Printf("Reading log events from %s", f.path)
	return NewFileLogSource(f.path)
}
