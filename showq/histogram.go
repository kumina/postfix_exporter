package showq

import "github.com/prometheus/client_golang/prometheus"

type Histograms struct {
	SizeHistogram prometheus.ObserverVec
	AgeHistogram  prometheus.ObserverVec
}

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
				Buckets:   []float64{1e1, 1e2, 1e3, 1e4, 1e5, 1e6, 1e7, 1e8},
			},
			[]string{"queue"}),
	}
	// Initialize all queue buckets to zero.
	for _, q := range []string{queueActive, queueHold, queueOther, queueIncoming} {
		hist.SizeHistogram.WithLabelValues(q)
		hist.AgeHistogram.WithLabelValues(q)
	}
	return hist
}
