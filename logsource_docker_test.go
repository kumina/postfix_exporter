// +build !nodocker

package main

import (
	"context"
	"io"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/stretchr/testify/assert"
)

func TestNewDockerLogSource(t *testing.T) {
	ctx := context.Background()
	c := &fakeDockerClient{}
	src, err := NewDockerLogSource(ctx, c, "acontainer")
	if err != nil {
		t.Fatalf("NewDockerLogSource failed: %v", err)
	}

	assert.Equal(t, []string{"acontainer"}, c.containerLogsCalls, "A call to ContainerLogs should be made.")

	if err := src.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	assert.Equal(t, 1, c.closeCalls, "A call to Close should be made.")
}

func TestDockerLogSource_Path(t *testing.T) {
	ctx := context.Background()
	c := &fakeDockerClient{}
	src, err := NewDockerLogSource(ctx, c, "acontainer")
	if err != nil {
		t.Fatalf("NewDockerLogSource failed: %v", err)
	}
	defer src.Close()

	assert.Equal(t, "docker:acontainer", src.Path(), "Path should be set by New.")
}

func TestDockerLogSource_Read(t *testing.T) {
	ctx := context.Background()

	c := &fakeDockerClient{
		logsReader: ioutil.NopCloser(strings.NewReader("Feb 13 23:31:30 ahost anid[123]: aline\n")),
	}
	src, err := NewDockerLogSource(ctx, c, "acontainer")
	if err != nil {
		t.Fatalf("NewDockerLogSource failed: %v", err)
	}
	defer src.Close()

	s, err := src.Read(ctx)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	assert.Equal(t, "Feb 13 23:31:30 ahost anid[123]: aline", s, "Read should get data from the journal entry.")
}

type fakeDockerClient struct {
	logsReader io.ReadCloser

	containerLogsCalls []string
	closeCalls         int
}

func (c *fakeDockerClient) ContainerLogs(ctx context.Context, containerID string, opts types.ContainerLogsOptions) (io.ReadCloser, error) {
	c.containerLogsCalls = append(c.containerLogsCalls, containerID)
	return c.logsReader, nil
}

func (c *fakeDockerClient) Close() error {
	c.closeCalls++
	return nil
}
