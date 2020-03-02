// +build nosystemd !linux
// This file contains stubs to support non-systemd use

package logCollector

import (
	"io"

	"github.com/alecthomas/kingpin"
)

type Journal struct {
	io.Closer
	Path string
}

func SystemdFlags(enable *bool, unit, slice, path *string, app *kingpin.Application) {}

func NewJournal(unit, slice, path string) (*Journal, error) {
	return nil, nil
}

func (e *LogCollector) CollectLogfileFromJournal() error {
	return nil
}
