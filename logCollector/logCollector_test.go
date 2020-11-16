package logCollector

import (
	"testing"

	"github.com/kumina/postfix_exporter/testUtils"
	"github.com/prometheus/client_golang/prometheus"
)

func TestPostfixExporter_CollectFromLogline(t *testing.T) {
	type args struct {
		line           []string
		logUnsupported bool
	}

	type wanted struct {
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
		name   string
		args   args
		wanted wanted
	}{
		{
			name: "Invalid line",
			args: args{
				line: []string{
					"Whatever",
				},
			},
			wanted: wanted{
				unsupportedLines: 1,
			},
		},
		{
			name: "Single line",
			args: args{
				line: []string{
					"Feb 11 16:49:24 letterman postfix/qmgr[8204]: AAB4D259B1: removed",
				},
			},
			wanted: wanted{
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
			},
			wanted: wanted{
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
			},
			wanted: wanted{
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
			},
			wanted: wanted{
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
			},
			wanted: wanted{
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
			},
			wanted: wanted{
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
			},
			wanted: wanted{
				outgoingTLS: 2,
			},
		},
		{
			name: "Testing delays",
			args: args{
				line: []string{
					"Feb 24 16:18:40 letterman postfix/smtp[59649]: 5270320179: to=<hebj@telia.com>, relay=mail.telia.com[81.236.60.210]:25, delay=2017, delays=0.1/2017/0.03/0.05, dsn=2.0.0, status=sent (250 2.0.0 6FVIjIMwUJwU66FVIjAEB0 mail accepted for delivery)",
				},
			},
			wanted: wanted{
				smtpDelays:      4,
				smtpStatusCount: 1,
			},
		},
		{
			name: "delivery status",
			args: args{line: []string{
				"Mar 14 02:19:56 letterman postfix/smtp[39982]: 7452832B28: to=<example@yahoo.com>, relay=mta6.am0.yahoodns.net[67.195.228.94]:25, delay=1905, delays=1682/217/1.2/5.6, dsn=2.0.0, status=sent (250 ok dirdel)",
				"Mar 14 02:19:56 letterman postfix/smtp[42367]: E7D8726A3B: to=<examplet@gmail.com>, relay=gmail-smtp-in.l.google.com[108.177.14.27]:25, delay=2134, delays=2133/0/0.12/0.59, dsn=2.0.0, status=sent (250 2.0.0 OK  1584148796 q7si7179",
				"Mar 14 03:25:38 letterman postfix/smtp[6656]: 5BECF63C4A: to=<example@icloud.se>, relay=none, delay=44899, delays=44869/0/30/0, dsn=4.4.1, status=deferred (connect to icloud.se[17.253.142.4]:25: Connection timed out)",
				"Mar 14 03:25:56 letterman postfix/smtp[5252]: 5412634DDB: to=<example@gafe.se>, relay=none, delay=57647, delays=57647/0/0/0, dsn=4.4.1, status=deferred (connect to gafe.se[151.252.30.111]:25: Connection refused)",
				"Mar 14 02:44:00 letterman postfix/smtp[53185]: BB40124D8D: to=<example@hotmai.com>, relay=none, delay=3804, delays=3804/0/0.01/0, dsn=5.4.4, status=bounced (Host or domain name not found. Name service error for name=hotmai.com type=A: Host found but no data record of requested type)",
			},
			},
			wanted: wanted{
				smtpDelays:      20,
				smtpStatusCount: 5,
			},
		},
		{
			name: "Postfix 3 logs",
			args: args{
				line: []string{
					"2020-03-19T09:29:04.507835+00:00 638fbdd7968d postfix/smtp[156]: CA1AD20189: lost connection with mta6.am0.yahoodns.net[67.195.204.79] while sending RCPT TO",
					"2020-03-19T09:29:05.519970+00:00 638fbdd7968d postfix/smtp[156]: Trusted TLS connection established to mta5.am0.yahoodns.net[67.195.228.106]:25: TLSv1.2 with cipher ECDHE-RSA-AES128-GCM-SHA256 (128/128 bits)",
					"2020-03-19T09:29:05.858036+00:00 638fbdd7968d postfix/smtp[156]: CA1AD20189: to=<example@yahoo.com>, relay=mta5.am0.yahoodns.net[67.195.228.106]:25, delay=69785, delays=69745/38/2.3/0.17, dsn=4.7.0, status=deferred (host mta5.am0.yahoodns.net[67.195.228.106] said: 421 4.7.0 [TSS04] Messages from 192.36.214.188 temporarily deferred due to user complaints - 4.16.55.1; see https://help.yahoo.com/kb/postmaster/SLN3434.html (in reply to MAIL FROM command))",
					"2020-03-19T09:29:07.526924+00:00 638fbdd7968d postfix/smtp[158]: Trusted TLS connection established to mta5.am0.yahoodns.net[67.195.204.77]:25: TLSv1.2 with cipher ECDHE-RSA-AES128-GCM-SHA256 (128/128 bits)",
					"2020-03-19T09:29:07.746143+00:00 638fbdd7968d postfix/smtp[158]: 973D120142: host mta5.am0.yahoodns.net[67.195.204.77] said: 421 4.7.0 [TSS04] Messages from 192.36.214.188 temporarily deferred due to user complaints - 4.16.55.1; see https://help.yahoo.com/kb/postmaster/SLN3434.html (in reply to MAIL FROM command)",
					"2020-03-19T09:29:07.746330+00:00 638fbdd7968d postfix/smtp[158]: 973D120142: lost connection with mta5.am0.yahoodns.net[67.195.204.77] while sending RCPT TO",
					"2020-03-19T09:29:08.613838+00:00 638fbdd7968d postfix/smtp[158]: Trusted TLS connection established to mta6.am0.yahoodns.net[67.195.204.77]:25: TLSv1.2 with cipher ECDHE-RSA-AES128-GCM-SHA256 (128/128 bits)",
					"2020-03-19T09:29:08.831955+00:00 638fbdd7968d postfix/smtp[158]: 973D120142: to=<example@yahoo.com>, relay=mta6.am0.yahoodns.net[67.195.204.77]:25, delay=69419, delays=69376/41/1.9/0.11, dsn=4.7.0, status=deferred (host mta6.am0.yahoodns.net[67.195.204.77] said: 421 4.7.0 [TSS04] Messages from 192.36.214.188 temporarily deferred due to user complaints - 4.16.55.1; see https://help.yahoo.com/kb/postmaster/SLN3434.html (in reply to MAIL FROM command))",
					"2020-03-19T09:29:27.657686+00:00 638fbdd7968d postfix/smtpd[87]: connect from localhost[127.0.0.1]",
					"2020-03-19T09:29:27.658959+00:00 638fbdd7968d postfix/smtpd[87]: disconnect from localhost[127.0.0.1] ehlo=1 quit=1 commands=2",
				},
			},
			wanted: wanted{
				outgoingTLS:      3,
				smtpdDisconnects: 1,
				unsupportedLines: 3,
				smtpDelays:       2 * 4,
				smtpdConnects:    1,
				smtpStatusCount:  2,
			},
		},
		{
			name: "tlsproxy counters",
			args: args{
				line: []string{
					"2020-03-19T16:16:47.883506+00:00 638fbdd7968d postfix/tlsproxy[26371]: DISCONNECT [209.85.233.27]:25",
					"2020-03-19T16:16:47.986953+00:00 638fbdd7968d postfix/tlsproxy[26371]: CONNECT to [173.194.222.27]:25",
					"2020-03-19T16:16:48.000693+00:00 638fbdd7968d postfix/tlsproxy[26371]: Trusted TLS connection established to gmail-smtp-in.l.google.com[173.194.222.27]:25: TLSv1.3 with cipher TLS_AES_256_GCM_SHA384 (256/256 bits) key-exchange X25519 server-signature RSA-PSS (2048 bits)",
					"2020-03-19T16:16:48.920090+00:00 638fbdd7968d postfix/tlsproxy[26371]: CONNECT to [93.188.3.14]:25",
					"2020-03-19T16:16:48.929901+00:00 638fbdd7968d postfix/tlsproxy[26371]: Trusted TLS connection established to mailcluster.loopia.se[93.188.3.14]:25: TLSv1.3 with cipher TLS_AES_256_GCM_SHA384 (256/256 bits) key-exchange X25519 server-signature RSA-PSS (2048 bits) server-digest SHA256",
					"2020-03-19T16:16:49.070051+00:00 638fbdd7968d postfix/tlsproxy[26371]: CONNECT to [173.194.222.27]:25",
					"2020-03-19T16:16:49.083360+00:00 638fbdd7968d postfix/tlsproxy[26371]: Trusted TLS connection established to gmail-smtp-in.l.google.com[173.194.222.27]:25: TLSv1.3 with cipher TLS_AES_256_GCM_SHA384 (256/256 bits) key-exchange X25519 server-signature RSA-PSS (2048 bits)",
					"2020-03-19T16:16:49.214908+00:00 638fbdd7968d postfix/tlsproxy[26371]: CONNECT to [212.3.0.44]:25",
					"2020-03-19T16:16:49.242338+00:00 638fbdd7968d postfix/tlsproxy[26371]: Anonymous TLS connection established to mail4.gotanet.se[212.3.0.44]:25: TLSv1 with cipher ADH-AES256-SHA (256/256 bits)",
					"2020-03-19T16:16:49.329134+00:00 638fbdd7968d postfix/tlsproxy[26371]: DISCONNECT [212.3.0.44]:25",
					"2020-03-19T16:16:49.345054+00:00 638fbdd7968d postfix/tlsproxy[26371]: DISCONNECT [93.188.3.14]:25",
					"2020-03-19T16:16:49.383853+00:00 638fbdd7968d postfix/tlsproxy[26371]: CONNECT to [104.47.8.36]:25",
					"2020-03-19T16:16:49.513395+00:00 638fbdd7968d postfix/tlsproxy[26371]: Trusted TLS connection established to stenungsund-se.mail.protection.outlook.com[104.47.8.36]:25: TLSv1.2 with cipher ECDHE-RSA-AES256-GCM-SHA384 (256/256 bits)",
					"2020-03-19T16:16:49.592066+00:00 638fbdd7968d postfix/tlsproxy[26371]: CONNECT to [173.194.222.27]:25",
					"2020-03-19T16:16:49.609308+00:00 638fbdd7968d postfix/tlsproxy[26371]: Trusted TLS connection established to gmail-smtp-in.l.google.com[173.194.222.27]:25: TLSv1.3 with cipher TLS_AES_256_GCM_SHA384 (256/256 bits) key-exchange X25519 server-signature RSA-PSS (2048 bits)",
					"2020-03-19T16:16:49.683983+00:00 638fbdd7968d postfix/tlsproxy[26371]: CONNECT to [173.194.222.27]:25",
					"2020-03-19T16:16:49.698182+00:00 638fbdd7968d postfix/tlsproxy[26371]: Trusted TLS connection established to gmail-smtp-in.l.google.com[173.194.222.27]:25: TLSv1.3 with cipher TLS_AES_256_GCM_SHA384 (256/256 bits) key-exchange X25519 server-signature RSA-PSS (2048 bits)",
					"2020-03-19T16:16:49.765462+00:00 638fbdd7968d postfix/tlsproxy[26371]: DISCONNECT [173.194.222.27]:25",
				},
				logUnsupported: true,
			},
			wanted: wanted{},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gauge := prometheus.NewGauge(prometheus.GaugeOpts{})
			e, err := NewLogCollector(tt.args.logUnsupported, gauge)
			if err != nil {
				t.Fatalf("cannot construct collector: %v", err)
			}
			for _, line := range tt.args.line {
				e.CollectFromLogLine(line)
			}
			testUtils.AssertCounterEquals(t, e.cleanupProcesses, tt.wanted.cleanupProcesses, "Wrong number of cleanupProcesses processed")
			testUtils.AssertCounterEquals(t, e.cleanupRejects, tt.wanted.cleanupRejects, "Wrong number of cleanupRejects processed")
			testUtils.AssertCounterEquals(t, e.cleanupNotAccepted, tt.wanted.cleanupNotAccepted, "Wrong number of cleanupNotAccepted processed")
			testUtils.AssertCounterEquals(t, e.lmtpDelays, tt.wanted.lmtpDelays, "Wrong number of lmtpDelays processed")
			testUtils.AssertCounterEquals(t, e.opendkimSignatureAdded, tt.wanted.opendkimSignatureAdded, "Wrong number of opendkimSignatureAdded processed")
			testUtils.AssertCounterEquals(t, e.pipeDelays, tt.wanted.pipeDelays, "Wrong number of pipeDelays processed")
			testUtils.AssertCounterEquals(t, e.postfixUp, tt.wanted.postfixUp, "Wrong number of postfixUp processed")
			testUtils.AssertCounterEquals(t, e.qmgrInsertsNrcpt, tt.wanted.qmgrInsertsNrcpt, "Wrong number of qmgrInsertsNrcpt processed")
			testUtils.AssertCounterEquals(t, e.qmgrInsertsSize, tt.wanted.qmgrInsertsSize, "Wrong number of qmgrInsertsSize processed")
			testUtils.AssertCounterEquals(t, e.qmgrRemoves, tt.wanted.removedCount, "Wrong number of removedCount processed")
			testUtils.AssertCounterEquals(t, e.smtpDelays, tt.wanted.smtpDelays, "Wrong number of smtpDelays processed")
			testUtils.AssertCounterEquals(t, e.smtpTLSConnects, tt.wanted.outgoingTLS, "Wrong number of TLS connections counted")
			testUtils.AssertCounterEquals(t, e.smtpConnectionTimedOut, tt.wanted.smtpConnectionTimedOut, "Wrong number of smtpConnectionTimedOut processed")
			testUtils.AssertCounterEquals(t, e.smtpdConnects, tt.wanted.smtpdConnects, "Wrong number of smtpdConnects processed")
			testUtils.AssertCounterEquals(t, e.smtpdDisconnects, tt.wanted.smtpdDisconnects, "Wrong number of smtpdDisconnects processed")
			testUtils.AssertCounterEquals(t, e.smtpdFCrDNSErrors, tt.wanted.smtpdFCrDNSErrors, "Wrong number of smtpdFCrDNSErrors processed")
			testUtils.AssertCounterEquals(t, e.smtpdLostConnections, tt.wanted.smtpdLostConnections, "Wrong number of smtpdLostConnections processed")
			testUtils.AssertCounterEquals(t, e.smtpdProcesses, tt.wanted.smtpdMessagesProcessed, "Wrong number of smtpdMessagesProcessed processed")
			testUtils.AssertCounterEquals(t, e.smtpdRejects, tt.wanted.smtpdRejects, "Wrong number of smtpdRejects processed")
			testUtils.AssertCounterEquals(t, e.smtpdSASLAuthenticationFailures, tt.wanted.saslFailedCount, "Wrong number of Sasl counter counted")
			testUtils.AssertCounterEquals(t, e.smtpdTLSConnects, tt.wanted.smtpdTLSConnects, "Wrong number of smtpdTLSConnects processed")
			testUtils.AssertCounterEquals(t, e.smtpStatusCount, tt.wanted.smtpStatusCount, "Wrong number of smtpStatusCount processed")
			testUtils.AssertCounterEquals(t, e.unsupportedLogEntries, tt.wanted.unsupportedLines, "Wrong number of unsupported messages processed")
		})
	}
}
