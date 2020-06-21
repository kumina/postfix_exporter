package main

import (
	"context"
	"fmt"
	"io"

	"github.com/alecthomas/kingpin"
)

// A LogSourceFactory provides a repository of log sources that can be
// instantiated from command line flags.
type LogSourceFactory interface {
	// Init adds the factory's struct fields as flags in the
	// application.
	Init(*kingpin.Application)

	// New attempts to create a new log source. This is called after
	// flags have been parsed. Returning `nil, nil`, means the user
	// didn't want this log source.
	New(context.Context) (LogSourceCloser, error)
}

type LogSourceCloser interface {
	io.Closer
	LogSource
}

var logSourceFactories []LogSourceFactory

// RegisterLogSourceFactory can be called from module `init` functions
// to register factories.
func RegisterLogSourceFactory(lsf LogSourceFactory) {
	logSourceFactories = append(logSourceFactories, lsf)
}

// InitLogSourceFactories runs Init on all factories. The
// initialization order is arbitrary, except `fileLogSourceFactory` is
// always last (the fallback). The file log source must be last since
// it's enabled by default.
func InitLogSourceFactories(app *kingpin.Application) {
	RegisterLogSourceFactory(&fileLogSourceFactory{})

	for _, f := range logSourceFactories {
		f.Init(app)
	}
}

// NewLogSourceFromFactories iterates through the factories and
// attempts to instantiate a log source. The first factory to return
// success wins.
func NewLogSourceFromFactories(ctx context.Context) (LogSourceCloser, error) {
	for _, f := range logSourceFactories {
		src, err := f.New(ctx)
		if err != nil {
			return nil, err
		}
		if src != nil {
			return src, nil
		}
	}

	return nil, fmt.Errorf("no log source configured")
}
