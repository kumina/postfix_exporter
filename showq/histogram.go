package showq

import "github.com/prometheus/client_golang/prometheus"

type Histograms struct {
	SizeHistogram prometheus.ObserverVec
	AgeHistogram  prometheus.ObserverVec
}

const (
	minute = 60
	hour   = 60 * minute
	day    = 24 * hour
)

func NewHistograms() Histograms {
	hist := Histograms{
		SizeHistogram: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "postfix",
				Name:      "showq_message_size_bytes",
				Help:      "Size of messages in Postfix's message queue, in bytes",
				Buckets:   []float64{1e3, 1e4, 1e5, 1e6, 1e7, 1e8, 1e9},
			},
			[]string{"queue"}),
		AgeHistogram: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "postfix",
				Name:      "showq_message_age_seconds",
				Help:      "Age of messages in Postfix's message queue, in seconds",
				Buckets:   []float64{1, minute, 5 * minute, 10 * minute, hour, hour * 6, hour * 12, hour * 18, 1 * day, 2 * day},
			},
			[]string{"queue"}),
	}
	// Initialize all queue buckets to zero.
	for _, q := range []string{queueActive, queueHold, queueOther, queueIncoming, queueDeferred} {
		hist.SizeHistogram.WithLabelValues(q)
		hist.AgeHistogram.WithLabelValues(q)
	}
	return hist
}
