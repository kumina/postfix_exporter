package logCollector

import (
	"fmt"
	"strings"

	"github.com/kumina/postfix_exporter/testUtils"
	"github.com/prometheus/client_golang/prometheus"
)

func ExampleLogCollector_CollectFromLogLine() {
	lines := `
Mar 14 02:19:56 letterman postfix/smtp[39982]: 7452832B28: to=<example@yahoo.com>, relay=mta6.am0.yahoodns.net[67.195.228.94]:25, delay=1905, delays=1682/217/1.2/5.6, dsn=2.0.0, status=sent (250 ok dirdel)
Mar 14 02:19:56 letterman postfix/smtp[42367]: E7D8726A3B: to=<examplet@gmail.com>, relay=gmail-smtp-in.l.google.com[108.177.14.27]:25, delay=2134, delays=2133/0/0.12/0.59, dsn=2.0.0, status=sent (250 2.0.0 OK  1584148796 q7si7179
Mar 14 03:25:38 letterman postfix/smtp[6656]: 5BECF63C4A: to=<example@icloud.se>, relay=none, delay=44899, delays=44869/0/30/0, dsn=4.4.1, status=deferred (connect to icloud.se[17.253.142.4]:25: Connection timed out)
`
	gauge := prometheus.NewGauge(prometheus.GaugeOpts{})
	e, err := NewLogCollector(false, gauge)
	if err != nil {
		fmt.Printf("cannot construct collector: %v", err)
		return
	}
	for _, line := range strings.Split(lines, "\n") {
		e.CollectFromLogLine(line)
	}
	collectors := []prometheus.Collector{e.smtpDelays, e.qmgrInsertsNrcpt, e.qmgrInsertsSize,
		e.cleanupProcesses, e.unsupportedLogEntries, e.opendkimSignatureAdded, e.smtpConnectionTimedOut,
		e.qmgrRemoves, e.cleanupNotAccepted, e.cleanupRejects,
	}
	for _, collector := range collectors {
		count, desc, err := testUtils.GetCount(collector)
		if err != nil {
			fmt.Printf("failed to get count for %v: %v\n", desc, err)
		}
		fmt.Printf("%v: %v\n", desc, count)
	}
	//  Output:
	//[Desc{fqName: "postfix_smtp_delivery_delay_seconds", help: "SMTP message processing time in seconds.", constLabels: {}, variableLabels: [stage]}]: 12
	//[Desc{fqName: "postfix_qmgr_messages_inserted_receipients", help: "Number of receipients per message inserted into the mail queues.", constLabels: {}, variableLabels: []}]: 0
	//[Desc{fqName: "postfix_qmgr_messages_inserted_size_bytes", help: "Size of messages inserted into the mail queues in bytes.", constLabels: {}, variableLabels: []}]: 0
	//[Desc{fqName: "postfix_cleanup_messages_processed_total", help: "Total number of messages processed by cleanup.", constLabels: {}, variableLabels: []}]: 0
	//[Desc{fqName: "postfix_unsupported_log_entries_total", help: "Log entries that could not be processed.", constLabels: {}, variableLabels: [service]}]: 2
	//[Desc{fqName: "opendkim_signatures_added_total", help: "Total number of messages signed.", constLabels: {}, variableLabels: [subject domain]}]: 0
	//[Desc{fqName: "postfix_smtp_connection_timed_out_total", help: "Total number of messages that have been deferred on SMTP.", constLabels: {}, variableLabels: []}]: 0
	//[Desc{fqName: "postfix_qmgr_messages_removed_total", help: "Total number of messages removed from mail queues.", constLabels: {}, variableLabels: []}]: 0
	//[Desc{fqName: "postfix_cleanup_messages_not_accepted_total", help: "Total number of messages not accepted by cleanup.", constLabels: {}, variableLabels: []}]: 0
	//[Desc{fqName: "postfix_cleanup_messages_rejected_total", help: "Total number of messages rejected by cleanup.", constLabels: {}, variableLabels: []}]: 0
}
