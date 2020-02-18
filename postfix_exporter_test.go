package main

import (
	"github.com/hpcloud/tail"
	"github.com/prometheus/client_golang/prometheus"
	io_prometheus_client "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestPostfixExporter_CollectFromLogline(t *testing.T) {
	type fields struct {
		showqPath                       string
		journal                         *Journal
		tailer                          *tail.Tail
		cleanupProcesses                prometheus.Counter
		cleanupRejects                  prometheus.Counter
		cleanupNotAccepted              prometheus.Counter
		lmtpDelays                      *prometheus.HistogramVec
		pipeDelays                      *prometheus.HistogramVec
		qmgrInsertsNrcpt                prometheus.Histogram
		qmgrInsertsSize                 prometheus.Histogram
		qmgrRemoves                     prometheus.Counter
		smtpDelays                      *prometheus.HistogramVec
		smtpTLSConnects                 *prometheus.CounterVec
		smtpDeferreds                   prometheus.Counter
		smtpdConnects                   prometheus.Counter
		smtpdDisconnects                prometheus.Counter
		smtpdFCrDNSErrors               prometheus.Counter
		smtpdLostConnections            *prometheus.CounterVec
		smtpdProcesses                  *prometheus.CounterVec
		smtpdRejects                    *prometheus.CounterVec
		smtpdSASLAuthenticationFailures prometheus.Counter
		smtpdTLSConnects                *prometheus.CounterVec
		unsupportedLogEntries           *prometheus.CounterVec
	}
	type args struct {
		line  []string
		count int
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "Single line",
			args: args{
				line: []string{
					"Feb 11 16:49:24 letterman postfix/qmgr[8204]: AAB4D259B1: removed",
				},
				count: 1,
			},
		},
		{
			name: "Multiple lines",
			args: args{
				line: []string{
					"Feb 11 16:49:24 letterman postfix/qmgr[8204]: AAB4D259B1: removed",
					"Feb 11 16:49:24 letterman postfix/qmgr[8204]: C2032259E6: removed",
					"Feb 11 16:49:24 letterman postfix/qmgr[8204]: B83C4257DC: removed",
					"Feb 11 16:49:24 letterman postfix/qmgr[8204]: 721BE256EA: removed",
					"Feb 11 16:49:25 letterman postfix/qmgr[8204]: CA94A259EB: removed",
					"Feb 11 16:49:25 letterman postfix/qmgr[8204]: AC1E3259E1: removed",
					"Feb 11 16:49:25 letterman postfix/qmgr[8204]: D114D221E3: removed",
					"Feb 11 16:49:25 letterman postfix/qmgr[8204]: A55F82104D: removed",
					"Feb 11 16:49:25 letterman postfix/qmgr[8204]: D6DAA259BC: removed",
					"Feb 11 16:49:25 letterman postfix/qmgr[8204]: E3908259F0: removed",
					"Feb 11 16:49:25 letterman postfix/qmgr[8204]: 0CBB8259BF: removed",
					"Feb 11 16:49:25 letterman postfix/qmgr[8204]: EA3AD259F2: removed",
					"Feb 11 16:49:25 letterman postfix/qmgr[8204]: DDEF824B48: removed",
					"Feb 11 16:49:26 letterman postfix/qmgr[8204]: 289AF21DB9: removed",
					"Feb 11 16:49:26 letterman postfix/qmgr[8204]: 6192B260E8: removed",
					"Feb 11 16:49:26 letterman postfix/qmgr[8204]: F2831259F4: removed",
					"Feb 11 16:49:26 letterman postfix/qmgr[8204]: 09D60259F8: removed",
					"Feb 11 16:49:26 letterman postfix/qmgr[8204]: 13A19259FA: removed",
					"Feb 11 16:49:26 letterman postfix/qmgr[8204]: 2D42722065: removed",
					"Feb 11 16:49:26 letterman postfix/qmgr[8204]: 746E325A0E: removed",
					"Feb 11 16:49:26 letterman postfix/qmgr[8204]: 4D2F125A02: removed",
					"Feb 11 16:49:26 letterman postfix/qmgr[8204]: E30BC259EF: removed",
					"Feb 11 16:49:26 letterman postfix/qmgr[8204]: DC88924DA1: removed",
					"Feb 11 16:49:26 letterman postfix/qmgr[8204]: 2164B259FD: removed",
					"Feb 11 16:49:26 letterman postfix/qmgr[8204]: 8C30525A14: removed",
					"Feb 11 16:49:26 letterman postfix/qmgr[8204]: 8DCCE25A15: removed",
					"Feb 11 16:49:26 letterman postfix/qmgr[8204]: C5217255D5: removed",
					"Feb 11 16:49:27 letterman postfix/qmgr[8204]: D8EE625A28: removed",
					"Feb 11 16:49:27 letterman postfix/qmgr[8204]: 9AD7C25A19: removed",
					"Feb 11 16:49:27 letterman postfix/qmgr[8204]: D0EEE2596C: removed",
					"Feb 11 16:49:27 letterman postfix/qmgr[8204]: DFE732172E: removed",
				},
				count: 31,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &PostfixExporter{
				showqPath:                       tt.fields.showqPath,
				journal:                         tt.fields.journal,
				tailer:                          tt.fields.tailer,
				cleanupProcesses:                tt.fields.cleanupProcesses,
				cleanupRejects:                  tt.fields.cleanupRejects,
				cleanupNotAccepted:              tt.fields.cleanupNotAccepted,
				lmtpDelays:                      tt.fields.lmtpDelays,
				pipeDelays:                      tt.fields.pipeDelays,
				qmgrInsertsNrcpt:                tt.fields.qmgrInsertsNrcpt,
				qmgrInsertsSize:                 tt.fields.qmgrInsertsSize,
				qmgrRemoves:                     tt.fields.qmgrRemoves,
				smtpDelays:                      tt.fields.smtpDelays,
				smtpTLSConnects:                 tt.fields.smtpTLSConnects,
				smtpDeferreds:                   tt.fields.smtpDeferreds,
				smtpdConnects:                   tt.fields.smtpdConnects,
				smtpdDisconnects:                tt.fields.smtpdDisconnects,
				smtpdFCrDNSErrors:               tt.fields.smtpdFCrDNSErrors,
				smtpdLostConnections:            tt.fields.smtpdLostConnections,
				smtpdProcesses:                  tt.fields.smtpdProcesses,
				smtpdRejects:                    tt.fields.smtpdRejects,
				smtpdSASLAuthenticationFailures: tt.fields.smtpdSASLAuthenticationFailures,
				smtpdTLSConnects:                tt.fields.smtpdTLSConnects,
				unsupportedLogEntries:           tt.fields.unsupportedLogEntries,
			}
			counter := testCounter{}
			e.qmgrRemoves = &counter
			for _, line := range tt.args.line {
				e.CollectFromLogLine(line)
			}
			assert.Equal(t, tt.args.count, counter.Count(), "Wrong number of lines counted")
			if counter.Count() != tt.args.count {
				t.Fatal("Counter wrong: ")
			}
		})
	}
}

type testCounter struct {
	count int
}

func (t *testCounter) setCount(count int) {
	t.count = count
}

func (t *testCounter) Count() int {
	return t.count
}

func (t *testCounter) Add(add float64) {
}
func (t *testCounter) Collect(c chan<- prometheus.Metric) {
}
func (t *testCounter) Describe(c chan<- *prometheus.Desc) {
}
func (t *testCounter) Desc() *prometheus.Desc {
	return nil
}
func (t *testCounter) Inc() {
	t.count++
}
func (t *testCounter) Write(x *io_prometheus_client.Metric) error {
	return nil
}
