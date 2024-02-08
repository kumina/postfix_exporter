// +build !nodocker

package main

import (
	"bufio"
	"context"
	"io"
	"log"
	"strings"
	"errors"

	"github.com/alecthomas/kingpin"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

// A DockerLogSource reads log records from the given Docker
// journal.
type DockerLogSource struct {
	client      DockerClient
	sourceType  string
	sourceID    string
	reader      *bufio.Reader
}

// A DockerClient is the client interface that client.Client
// provides. See https://pkg.go.dev/github.com/docker/docker/client
type DockerClient interface {
	io.Closer
	ContainerLogs(context.Context, string, types.ContainerLogsOptions) (io.ReadCloser, error)
	ServiceLogs(context.Context, string, types.ContainerLogsOptions) (io.ReadCloser, error)
}

// NewDockerLogSource returns a log source for reading Docker logs.
func NewDockerLogSource(ctx context.Context, c DockerClient, sourceType string, sourceID string) (*DockerLogSource, error) {
	var r io.ReadCloser
	var err error
	if sourceType == "container" {
		r, err = c.ContainerLogs(ctx, sourceID, types.ContainerLogsOptions{
			ShowStdout: true,
			ShowStderr: true,
			Follow:     true,
			Tail:       "0",
		})
	} else if sourceType == "service" {
		r, err = c.ServiceLogs(ctx, sourceID, types.ContainerLogsOptions{
			ShowStdout: true,
			ShowStderr: true,
			Follow:     true,
			Tail:       "0",
		})
	} else {
		r, err = (io.ReadCloser)(nil), errors.New("docker.source.type should be set to either \"container\" or \"service\" when docker is enabled")
	}
	if err != nil {
		return nil, err
	}

	logSrc := &DockerLogSource{
		client:      c,
		sourceID:    sourceID,
		reader:      bufio.NewReader(r),
	}

	return logSrc, nil
}

func (s *DockerLogSource) Close() error {
	return s.client.Close()
}

func (s *DockerLogSource) Path() string {
	return "docker:" + s.sourceID
}

func (s *DockerLogSource) Read(ctx context.Context) (string, error) {
	line, err := s.reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

// A dockerLogSourceFactory is a factory that can create
// DockerLogSources from command line flags.
type dockerLogSourceFactory struct {
	enable      bool
	sourceType string
	sourceID string
}

func (f *dockerLogSourceFactory) Init(app *kingpin.Application) {
	app.Flag("docker.enable", "Read from Docker logs. Environment variable DOCKER_HOST can be used to change the address. See https://pkg.go.dev/github.com/docker/docker/client?tab=doc#NewEnvClient for more information.").Default("false").BoolVar(&f.enable)
	app.Flag("docker.source.type", "Source type for docker logs, \"conatiner\" or \"service\"").Default("container").StringVar(&f.sourceType)
	app.Flag("docker.source.id", "ID/name of the Postfix Docker container or service.").Default("postfix").StringVar(&f.sourceID)
}

func (f *dockerLogSourceFactory) New(ctx context.Context) (LogSourceCloser, error) {
	if !f.enable {
		return nil, nil
	}

	log.Println("Reading log events from Docker")
	c, err := client.NewEnvClient()
	if err != nil {
		return nil, err
	}
	return NewDockerLogSource(ctx, c, f.sourceType, f.sourceID)
}

func init() {
	RegisterLogSourceFactory(&dockerLogSourceFactory{})
}
