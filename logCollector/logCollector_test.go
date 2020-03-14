package logCollector

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	io_prometheus_client "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
)

func TestPostfixExporter_CollectFromLogline(t *testing.T) {
	type args struct {
		line                   []string
		removedCount           int
		saslFailedCount        int
		outgoingTLS            int
		smtpdMessagesProcessed int
		unsupportedLines       int
		cleanupProcesses       int
		cleanupRejects         int
		cleanupNotAccepted     int
		lmtpDelays             int
		pipeDelays             int
		qmgrInsertsNrcpt       int
		qmgrInsertsSize        int
		smtpDelays             int
		smtpConnectionTimedOut int
		smtpdConnects          int
		smtpdDisconnects       int
		smtpdFCrDNSErrors      int
		smtpdLostConnections   int
		smtpdRejects           int
		smtpdTLSConnects       int
		smtpStatusCount        int
		opendkimSignatureAdded int
		postfixUp              int
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "Invalid line",
			args: args{
				line: []string{
					"Whatever",
				},
				unsupportedLines: 1,
			},
		},
		{
			name: "Single line",
			args: args{
				line: []string{
					"Feb 11 16:49:24 letterman postfix/qmgr[8204]: AAB4D259B1: removed",
				},
				removedCount: 1,
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
				removedCount: 31,
			},
		},
		{
			name: "Qmgr sizes",
			args: args{
				line: []string{
					"Mar  6 01:13:50 letterman postfix/qmgr[1436]: D3120239C2: from=<sender@example.com>, size=5779, nrcpt=1 (queue active)",
					"Mar  6 01:13:50 letterman postfix/qmgr[1436]: 765162183F: from=<sender@example.com>, size=4042, nrcpt=1 (queue active)",
					"Mar  6 01:13:50 letterman postfix/qmgr[1436]: 765FE23A2C: from=<sender@example.com>, size=3978, nrcpt=1 (queue active)",
					"Mar  6 01:13:50 letterman postfix/qmgr[1436]: 7C14A2156B: from=<sender@example.com>, size=3985, nrcpt=1 (queue active)",
					"Mar  6 01:13:50 letterman postfix/qmgr[1436]: 816D523A36: from=<sender@example.com>, size=4067, nrcpt=1 (queue active)",
				},
				qmgrInsertsSize:  5,
				qmgrInsertsNrcpt: 5,
			},
		},
		{
			name: "SASL Failed",
			args: args{
				line: []string{
					"Apr 26 10:55:19 tcc1 postfix/smtpd[21126]: warning: SASL authentication failure: cannot connect to saslauthd server: Permission denied",
					"Apr 26 10:55:19 tcc1 postfix/smtpd[21126]: warning: SASL authentication failure: Password verification failed",
					"Apr 26 10:55:19 tcc1 postfix/smtpd[21126]: warning: laptop.local[192.168.1.2]: SASL PLAIN authentication failed: generic failure",
				},
				saslFailedCount:  1,
				unsupportedLines: 2,
			},
		},
		{
			name: "Cleanup lines processed",
			args: args{
				line: []string{
					"Mar  6 09:44:51 letterman postfix/cleanup[16382]: 3C3CE205E3: message-id=<1769646355.10117466.1583484291216@mail-processor-wa1>",
					"Mar  6 09:44:51 letterman postfix/cleanup[18638]: 3DDBF205E5: message-id=<1134533866.10117467.1583484291216@mail-processor-wa1>",
				},
				cleanupProcesses: 2,
			},
		},
		{
			name: "SASL login",
			args: args{
				line: []string{
					"Oct 30 13:19:26 mailgw-out1 postfix/smtpd[27530]: EB4B2C19E2: client=xxx[1.2.3.4], sasl_method=PLAIN, sasl_username=user@domain",
					"Feb 24 16:42:00 letterman postfix/smtpd[24906]: 1CF582025C: client=xxx[2.3.4.5]",
				},
				smtpdMessagesProcessed: 2,
			},
		},
		{
			name: "Issue #35",
			args: args{
				line: []string{
					"Jul 24 04:38:17 mail postfix/smtp[30582]: Verified TLS connection established to gmail-smtp-in.l.google.com[108.177.14.26]:25: TLSv1.3 with cipher TLS_AES_256_GCM_SHA384 (256/256 bits) key-exchange X25519 server-signature RSA-PSS (2048 bits) server-digest SHA256",
					"Jul 24 03:28:15 mail postfix/smtp[24052]: Verified TLS connection established to mx2.comcast.net[2001:558:fe21:2a::6]:25: TLSv1.2 with cipher ECDHE-RSA-AES256-GCM-SHA384 (256/256 bits)",
				},
				outgoingTLS: 2,
			},
		},
		{
			name: "Testing delays",
			args: args{
				line: []string{
					"Feb 24 16:18:40 letterman postfix/smtp[59649]: 5270320179: to=<hebj@telia.com>, relay=mail.telia.com[81.236.60.210]:25, delay=2017, delays=0.1/2017/0.03/0.05, dsn=2.0.0, status=sent (250 2.0.0 6FVIjIMwUJwU66FVIjAEB0 mail accepted for delivery)",
				},
				smtpDelays:      4,
				smtpStatusCount: 1,
			},
		},
		{name: "delivery status",
			args: args{line: []string{
				"Mar 14 02:19:56 letterman postfix/smtp[39982]: 7452832B28: to=<example@yahoo.com>, relay=mta6.am0.yahoodns.net[67.195.228.94]:25, delay=1905, delays=1682/217/1.2/5.6, dsn=2.0.0, status=sent (250 ok dirdel)",
				"Mar 14 02:19:56 letterman postfix/smtp[42367]: E7D8726A3B: to=<examplet@gmail.com>, relay=gmail-smtp-in.l.google.com[108.177.14.27]:25, delay=2134, delays=2133/0/0.12/0.59, dsn=2.0.0, status=sent (250 2.0.0 OK  1584148796 q7si7179",
				"Mar 14 03:25:38 letterman postfix/smtp[6656]: 5BECF63C4A: to=<example@icloud.se>, relay=none, delay=44899, delays=44869/0/30/0, dsn=4.4.1, status=deferred (connect to icloud.se[17.253.142.4]:25: Connection timed out)",
				"Mar 14 03:25:56 letterman postfix/smtp[5252]: 5412634DDB: to=<example@gafe.se>, relay=none, delay=57647, delays=57647/0/0/0, dsn=4.4.1, status=deferred (connect to gafe.se[151.252.30.111]:25: Connection refused)",
				"Mar 14 02:44:00 letterman postfix/smtp[53185]: BB40124D8D: to=<example@hotmai.com>, relay=none, delay=3804, delays=3804/0/0.01/0, dsn=5.4.4, status=bounced (Host or domain name not found. Name service error for name=hotmai.com type=A: Host found but no data record of requested type)",
			},
				smtpDelays:      20,
				smtpStatusCount: 5,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gaugeVec := prometheus.NewGaugeVec(prometheus.GaugeOpts{}, nil)
			e, err := NewLogCollector(false, gaugeVec)
			if err != nil {
				t.Fatalf("cannot construct collector: %v", err)
			}
			for _, line := range tt.args.line {
				e.CollectFromLogLine(line)
			}
			assertCounterEquals(t, e.cleanupProcesses, tt.args.cleanupProcesses, "Wrong number of cleanupProcesses processed")
			assertCounterEquals(t, e.cleanupRejects, tt.args.cleanupRejects, "Wrong number of cleanupRejects processed")
			assertCounterEquals(t, e.cleanupNotAccepted, tt.args.cleanupNotAccepted, "Wrong number of cleanupNotAccepted processed")
			assertCounterEquals(t, e.lmtpDelays, tt.args.lmtpDelays, "Wrong number of lmtpDelays processed")
			assertCounterEquals(t, e.opendkimSignatureAdded, tt.args.opendkimSignatureAdded, "Wrong number of opendkimSignatureAdded processed")
			assertCounterEquals(t, e.pipeDelays, tt.args.pipeDelays, "Wrong number of pipeDelays processed")
			assertCounterEquals(t, e.postfixUp, tt.args.postfixUp, "Wrong number of postfixUp processed")
			assertCounterEquals(t, e.qmgrInsertsNrcpt, tt.args.qmgrInsertsNrcpt, "Wrong number of qmgrInsertsNrcpt processed")
			assertCounterEquals(t, e.qmgrInsertsSize, tt.args.qmgrInsertsSize, "Wrong number of qmgrInsertsSize processed")
			assertCounterEquals(t, e.qmgrRemoves, tt.args.removedCount, "Wrong number of removedCount processed")
			assertCounterEquals(t, e.smtpDelays, tt.args.smtpDelays, "Wrong number of smtpDelays processed")
			assertCounterEquals(t, e.smtpTLSConnects, tt.args.outgoingTLS, "Wrong number of TLS connections counted")
			assertCounterEquals(t, e.smtpConnectionTimedOut, tt.args.smtpConnectionTimedOut, "Wrong number of smtpConnectionTimedOut processed")
			assertCounterEquals(t, e.smtpdConnects, tt.args.smtpdConnects, "Wrong number of smtpdConnects processed")
			assertCounterEquals(t, e.smtpdDisconnects, tt.args.smtpdDisconnects, "Wrong number of smtpdDisconnects processed")
			assertCounterEquals(t, e.smtpdFCrDNSErrors, tt.args.smtpdFCrDNSErrors, "Wrong number of smtpdFCrDNSErrors processed")
			assertCounterEquals(t, e.smtpdLostConnections, tt.args.smtpdLostConnections, "Wrong number of smtpdLostConnections processed")
			assertCounterEquals(t, e.smtpdProcesses, tt.args.smtpdMessagesProcessed, "Wrong number of smtpdMessagesProcessed processed")
			assertCounterEquals(t, e.smtpdRejects, tt.args.smtpdRejects, "Wrong number of smtpdRejects processed")
			assertCounterEquals(t, e.smtpdSASLAuthenticationFailures, tt.args.saslFailedCount, "Wrong number of Sasl counter counted")
			assertCounterEquals(t, e.smtpdTLSConnects, tt.args.smtpdTLSConnects, "Wrong number of smtpdTLSConnects processed")
			assertCounterEquals(t, e.smtpStatusCount, tt.args.smtpStatusCount, "Wrong number of smtpStatusCount processed")
			assertCounterEquals(t, e.unsupportedLogEntries, tt.args.unsupportedLines, "Wrong number of unsupported messages processed")
		})
	}
}
func assertCounterEquals(t *testing.T, counter prometheus.Collector, expected int, message string) {

	if counter != nil {
		switch counter.(type) {
		case *prometheus.CounterVec:
			counter := counter.(*prometheus.CounterVec)
			metricsChan := make(chan prometheus.Metric)
			go func() {
				counter.Collect(metricsChan)
				close(metricsChan)
			}()
			var count int = 0
			for metric := range metricsChan {
				metricDto := io_prometheus_client.Metric{}
				metric.Write(&metricDto)
				count += int(*metricDto.Counter.Value)
			}
			assert.Equal(t, expected, count, message)
		case prometheus.Counter:
			metricsChan := make(chan prometheus.Metric)
			go func() {
				counter.Collect(metricsChan)
				close(metricsChan)
			}()
			var count int = 0
			for metric := range metricsChan {
				metricDto := io_prometheus_client.Metric{}
				metric.Write(&metricDto)
				count += int(*metricDto.Counter.Value)
			}
			assert.Equal(t, expected, count, message)
		case *prometheus.HistogramVec:
			counter := counter.(*prometheus.HistogramVec)
			metricsChan := make(chan prometheus.Metric)
			go func() {
				counter.Collect(metricsChan)
				close(metricsChan)
			}()
			var count int = 0
			for metric := range metricsChan {
				metricDto := io_prometheus_client.Metric{}
				metric.Write(&metricDto)
				count += int(*metricDto.Histogram.SampleCount)
			}
			assert.Equal(t, expected, count, message)
		case prometheus.Histogram:
			metricsChan := make(chan prometheus.Metric)
			go func() {
				counter.Collect(metricsChan)
				close(metricsChan)
			}()
			var count int = 0
			for metric := range metricsChan {
				metricDto := io_prometheus_client.Metric{}
				metric.Write(&metricDto)
				count += int(*metricDto.Histogram.SampleCount)
			}
			assert.Equal(t, expected, count, message)
		case *prometheus.GaugeVec:
			counter := counter.(*prometheus.GaugeVec)
			metricsChan := make(chan prometheus.Metric)
			go func() {
				counter.Collect(metricsChan)
				close(metricsChan)
			}()
			var count int = 0
			for metric := range metricsChan {
				metricDto := io_prometheus_client.Metric{}
				metric.Write(&metricDto)
				count += int(*metricDto.Gauge.Value)
			}
			assert.Equal(t, expected, count, message)
		case prometheus.Gauge:
			metricsChan := make(chan prometheus.Metric)
			go func() {
				counter.Collect(metricsChan)
				close(metricsChan)
			}()
			var count int = 0
			for metric := range metricsChan {
				metricDto := io_prometheus_client.Metric{}
				metric.Write(&metricDto)
				count += int(*metricDto.Gauge.Value)
			}
			assert.Equal(t, expected, count, message)
		default:
			t.Fatalf("Type not implemented: %T", counter)
		}
	}
}
