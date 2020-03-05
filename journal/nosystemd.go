// +build nosystemd !linux
// This file contains stubs to support non-systemd use

package journal

import (
	"context"
	"io"

	"github.com/alecthomas/kingpin"
	"github.com/prometheus/client_golang/prometheus"
)

type Journal struct {
	io.Closer
	Path string
}

func (j *Journal) Describe(chan<- *prometheus.Desc) {
}

func (j *Journal) Collect(chan<- prometheus.Metric) {
}

func SystemdFlags(enable *bool, unit, slice, path *string, app *kingpin.Application) {}

func NewJournal(unit, slice, path string) (*Journal, error) {
	return nil, nil
}

func (j *Journal) CollectLogLinesFromJournal(context.Context) (<-chan string, error) {
	return nil, nil
}
