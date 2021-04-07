package main

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	io_prometheus_client "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
)

func TestPostfixExporter_CollectFromLogline(t *testing.T) {
	type fields struct {
		showqPath                       string
		logSrc                          LogSource
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
		smtpStatusDeferred              prometheus.Counter
		smtpProcesses                   *prometheus.CounterVec
		smtpdConnects                   prometheus.Counter
		smtpdDisconnects                prometheus.Counter
		smtpdFCrDNSErrors               prometheus.Counter
		smtpdLostConnections            *prometheus.CounterVec
		smtpdProcesses                  *prometheus.CounterVec
		smtpdRejects                    *prometheus.CounterVec
		smtpdSASLAuthenticationFailures prometheus.Counter
		smtpdTLSConnects                *prometheus.CounterVec
		bounceNonDelivery               prometheus.Counter
		virtualDelivered                prometheus.Counter
		unsupportedLogEntries           *prometheus.CounterVec
	}
	type args struct {
		line                   []string
		removedCount           int
		saslFailedCount        int
		outgoingTLS            int
		smtpdMessagesProcessed int
		smtpMessagesProcessed  int
		bounceNonDelivery  int
		virtualDelivered       int
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
				removedCount:    1,
				saslFailedCount: 0,
			},
			fields: fields{
				qmgrRemoves:           prometheus.NewCounter(prometheus.CounterOpts{}),
				unsupportedLogEntries: prometheus.NewCounterVec(prometheus.CounterOpts{}, []string{"process"}),
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
				removedCount:    31,
				saslFailedCount: 0,
			},
			fields: fields{
				qmgrRemoves:           prometheus.NewCounter(prometheus.CounterOpts{}),
				unsupportedLogEntries: prometheus.NewCounterVec(prometheus.CounterOpts{}, []string{"process"}),
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
				saslFailedCount: 1,
				removedCount:    0,
			},
			fields: fields{
				smtpdSASLAuthenticationFailures: prometheus.NewCounter(prometheus.CounterOpts{}),
				unsupportedLogEntries:           prometheus.NewCounterVec(prometheus.CounterOpts{}, []string{"process"}),
				smtpProcesses:                   prometheus.NewCounterVec(prometheus.CounterOpts{}, []string{"status"}),
			},
		},
		{
			name: "SASL login",
			args: args{
				line: []string{
					"Oct 30 13:19:26 mailgw-out1 postfix/smtpd[27530]: EB4B2C19E2: client=xxx[1.2.3.4], sasl_method=PLAIN, sasl_username=user@domain",
					"Feb 24 16:42:00 letterman postfix/smtpd[24906]: 1CF582025C: client=xxx[2.3.4.5]",
				},
				removedCount:           0,
				saslFailedCount:        0,
				outgoingTLS:            0,
				smtpdMessagesProcessed: 2,
			},
			fields: fields{
				unsupportedLogEntries: prometheus.NewCounterVec(prometheus.CounterOpts{}, []string{"process"}),
				smtpdProcesses:        prometheus.NewCounterVec(prometheus.CounterOpts{}, []string{"sasl_method"}),
			},
		},
		{
			name: "Issue #35",
			args: args{
				line: []string{
					"Jul 24 04:38:17 mail postfix/smtp[30582]: Verified TLS connection established to gmail-smtp-in.l.google.com[108.177.14.26]:25: TLSv1.3 with cipher TLS_AES_256_GCM_SHA384 (256/256 bits) key-exchange X25519 server-signature RSA-PSS (2048 bits) server-digest SHA256",
					"Jul 24 03:28:15 mail postfix/smtp[24052]: Verified TLS connection established to mx2.comcast.net[2001:558:fe21:2a::6]:25: TLSv1.2 with cipher ECDHE-RSA-AES256-GCM-SHA384 (256/256 bits)",
				},
				removedCount:    0,
				saslFailedCount: 0,
				outgoingTLS:     2,
				smtpdMessagesProcessed: 0,
			},
			fields: fields{
				unsupportedLogEntries: prometheus.NewCounterVec(prometheus.CounterOpts{}, []string{"process"}),
				smtpTLSConnects:       prometheus.NewCounterVec(prometheus.CounterOpts{}, []string{"Verified", "TLSv1.2", "ECDHE-RSA-AES256-GCM-SHA384", "256", "256"}),
				smtpProcesses:         prometheus.NewCounterVec(prometheus.CounterOpts{}, []string{"status"}),
			},
		},
		{
			name: "Testing delays",
			args: args{
				line: []string{
					"Feb 24 16:18:40 letterman postfix/smtp[59649]: 5270320179: to=<hebj@telia.com>, relay=mail.telia.com[81.236.60.210]:25, delay=2017, delays=0.1/2017/0.03/0.05, dsn=2.0.0, status=sent (250 2.0.0 6FVIjIMwUJwU66FVIjAEB0 mail accepted for delivery)",
				},
				removedCount:           0,
				saslFailedCount:        0,
				outgoingTLS:            0,
				smtpdMessagesProcessed: 0,
				smtpMessagesProcessed:  1,
			},
			fields: fields{
				smtpDelays: prometheus.NewHistogramVec(prometheus.HistogramOpts{}, []string{"stage"}),
				smtpProcesses: prometheus.NewCounterVec(prometheus.CounterOpts{}, []string{"status"}),
			},
		},
		{
			name: "Testing different smtp statuses",
			args: args{
				line: []string{
					"Dec 29 02:54:09 mail postfix/smtp[7648]: 732BB407C3: host mail.domain.com[1.1.1.1] said: 451 DT:SPM 163 mx13,P8CowECpNVM_oEVaenoEAQ--.23796S3 1514512449, please try again 15min later (in reply to end of DATA command)",
					"Dec 29 02:54:12 mail postfix/smtp[7648]: 732BB407C3: to=<redacted@domain.com>, relay=mail.domain.com[1.1.1.1]:25, delay=6.2, delays=0.1/0/5.2/0.87, dsn=4.0.0, status=deferred (host mail.domain.com[1.1.1.1] said: 451 DT:SPM 163 mx40,WsCowAAnEhlCoEVa5GjcAA--.20089S3 1514512452, please try again 15min later (in reply to end of DATA command))",
					"Dec 29 03:03:48 mail postfix/smtp[8492]: 732BB407C3: to=<redacted@domain.com>, relay=mail.domain.com[1.1.1.1]:25, delay=582, delays=563/16/1.7/0.81, dsn=5.0.0, status=bounced (host mail.domain.com[1.1.1.1] said: 554 DT:SPM 163 mx9,O8CowEDJVFKCokVaRhz+AA--.26016S3 1514513028,please see http://mail.domain.com/help/help_spam.htm?ip= (in reply to end of DATA command))",
					"Dec 29 03:03:48 mail postfix/bounce[9321]: 732BB407C3: sender non-delivery notification: 5DE184083C",
				},
				smtpMessagesProcessed:  2,
				bounceNonDelivery: 1,
			},
			fields: fields{
				unsupportedLogEntries: prometheus.NewCounterVec(prometheus.CounterOpts{}, []string{"process"}),
				smtpDelays: prometheus.NewHistogramVec(prometheus.HistogramOpts{}, []string{"stage"}),
				smtpStatusDeferred: prometheus.NewCounter(prometheus.CounterOpts{}),
				smtpProcesses: prometheus.NewCounterVec(prometheus.CounterOpts{}, []string{"status"}),
				bounceNonDelivery: prometheus.NewCounter(prometheus.CounterOpts{}),
			},
		},
		{
			name: "Testing virtual delivered",
			args: args{
				line: []string{
					"Apr  7 15:35:20 123-mail postfix/virtual[20235]: 199041033BE: to=<me@domain.fr>, relay=virtual, delay=0.08, delays=0.08/0/0/0, dsn=2.0.0, status=sent (delivered to maildir)",
				},
				virtualDelivered: 1,
			},
			fields: fields{
				virtualDelivered: prometheus.NewCounter(prometheus.CounterOpts{}),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &PostfixExporter{
				showqPath:                       tt.fields.showqPath,
				logSrc:                          tt.fields.logSrc,
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
				smtpStatusDeferred:              tt.fields.smtpStatusDeferred,
				smtpProcesses:                   tt.fields.smtpProcesses,
				smtpdConnects:                   tt.fields.smtpdConnects,
				smtpdDisconnects:                tt.fields.smtpdDisconnects,
				smtpdFCrDNSErrors:               tt.fields.smtpdFCrDNSErrors,
				smtpdLostConnections:            tt.fields.smtpdLostConnections,
				smtpdProcesses:                  tt.fields.smtpdProcesses,
				smtpdRejects:                    tt.fields.smtpdRejects,
				smtpdSASLAuthenticationFailures: tt.fields.smtpdSASLAuthenticationFailures,
				smtpdTLSConnects:                tt.fields.smtpdTLSConnects,
				bounceNonDelivery:               tt.fields.bounceNonDelivery,
				virtualDelivered:                tt.fields.virtualDelivered,
				unsupportedLogEntries:           tt.fields.unsupportedLogEntries,
				logUnsupportedLines:             true,
			}
			for _, line := range tt.args.line {
				e.CollectFromLogLine(line)
			}
			assertCounterEquals(t, e.qmgrRemoves, tt.args.removedCount, "Wrong number of lines counted")
			assertCounterEquals(t, e.smtpdSASLAuthenticationFailures, tt.args.saslFailedCount, "Wrong number of Sasl counter counted")
			assertCounterEquals(t, e.smtpTLSConnects, tt.args.outgoingTLS, "Wrong number of TLS connections counted")
			assertCounterEquals(t, e.smtpdProcesses, tt.args.smtpdMessagesProcessed, "Wrong number of smtpd messages processed")
			assertCounterEquals(t, e.smtpProcesses, tt.args.smtpMessagesProcessed, "Wrong number of smtp messages processed")
			assertCounterEquals(t, e.bounceNonDelivery, tt.args.bounceNonDelivery, "Wrong number of non delivery notifications")
			assertCounterEquals(t, e.virtualDelivered, tt.args.virtualDelivered, "Wrong number of delivered mails")
		})
	}
}
func assertCounterEquals(t *testing.T, counter prometheus.Collector, expected int, message string) {

	if counter != nil && expected > 0 {
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
		default:
			t.Fatal("Type not implemented")
		}
	}
}
