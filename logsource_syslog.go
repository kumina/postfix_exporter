package main

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/alecthomas/kingpin"
	"gopkg.in/mcuadros/go-syslog.v2"
)

type syslogLogSource struct {
	network string
	listen  string
	server  *syslog.Server
	lines   chan string
	channel syslog.LogPartsChannel
}

func NewSyslogLogSource(network string, listen string) (*syslogLogSource, error) {
	channel := make(syslog.LogPartsChannel)
	handler := syslog.NewChannelHandler(channel)

	server := syslog.NewServer()
	server.SetFormat(syslog.Automatic)
	server.SetHandler(handler)
	if network == "udp" {
		server.ListenUDP(listen)
	} else {
		server.ListenTCP(listen)
	}
	server.Boot()
	lines := make(chan string, 1024)

	go func() {
		for logParts := range channel {
			lines <- fmt.Sprintf("%s[0]: %s", logParts["tag"], logParts["content"])
		}
	}()

	return &syslogLogSource{network, listen, server, lines, channel}, nil
}

func (s *syslogLogSource) Close() error {
	defer close(s.channel)
	return s.server.Kill()
}

func (s *syslogLogSource) Path() string {
	return fmt.Sprintf("%s:%s", s.network, s.listen)
}

func (s *syslogLogSource) Read(ctx context.Context) (string, error) {
	select {
	case line, ok := <-s.lines:
		if !ok {
			return "", io.EOF
		}
		return line, nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

// A syslogLogSourceFactory is a factory than can create log sources
// from command line flags.
type syslogLogSourceFactory struct {
	network string
	listen  string
	enable  bool
}

func (f *syslogLogSourceFactory) Init(app *kingpin.Application) {
	app.Flag("syslog.listen", "Host and port to listen on for syslog messages.").Default("0.0.0.0:514").StringVar(&f.listen)
	app.Flag("syslog.network", "Network protocol to use (tcp/udp).").Default("udp").StringVar(&f.network)
	app.Flag("syslog.enable", "Enable the syslog server.").Default("true").BoolVar(&f.enable)
}

func (f *syslogLogSourceFactory) New(ctx context.Context) (LogSourceCloser, error) {
	if !f.enable {
		return nil, nil
	}

	if !(f.network == "tcp" || f.network == "udp") {
		return nil, errors.New(fmt.Sprintf("Unknown network protocol %s", f.network))
	}

	return NewSyslogLogSource(f.network, f.listen)
}

func init() {
	RegisterLogSourceFactory(&syslogLogSourceFactory{})
}
