// +build !nodocker

package main

import (
	"bufio"
	"context"
	"io"
	"log"
	"strings"

	"github.com/alecthomas/kingpin"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

// A DockerLogSource reads log records from the given Docker
// journal.
type DockerLogSource struct {
	client      DockerClient
	containerID string
	reader      *bufio.Reader
}

// A DockerClient is the client interface that client.Client
// provides. See https://pkg.go.dev/github.com/docker/docker/client
type DockerClient interface {
	io.Closer
	ContainerLogs(context.Context, string, types.ContainerLogsOptions) (io.ReadCloser, error)
}

// NewDockerLogSource returns a log source for reading Docker logs.
func NewDockerLogSource(ctx context.Context, c DockerClient, containerID string) (*DockerLogSource, error) {
	r, err := c.ContainerLogs(ctx, containerID, types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
		Tail:       "0",
	})
	if err != nil {
		return nil, err
	}

	logSrc := &DockerLogSource{
		client:      c,
		containerID: containerID,
		reader:      bufio.NewReader(r),
	}

	return logSrc, nil
}

func (s *DockerLogSource) Close() error {
	return s.client.Close()
}

func (s *DockerLogSource) Path() string {
	return "docker:" + s.containerID
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
	containerID string
}

func (f *dockerLogSourceFactory) Init(app *kingpin.Application) {
	app.Flag("docker.enable", "Read from Docker logs. Environment variable DOCKER_HOST can be used to change the address. See https://pkg.go.dev/github.com/docker/docker/client?tab=doc#NewEnvClient for more information.").Default("false").BoolVar(&f.enable)
	app.Flag("docker.container.id", "ID/name of the Postfix Docker container.").Default("postfix").StringVar(&f.containerID)
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
	return NewDockerLogSource(ctx, c, f.containerID)
}

func init() {
	RegisterLogSourceFactory(&dockerLogSourceFactory{})
}
