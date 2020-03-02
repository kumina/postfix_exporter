package mock

import "github.com/prometheus/client_golang/prometheus"

type GaugeVec struct {
}

func (m GaugeVec) Describe(chan<- *prometheus.Desc) {
	panic("implement me")
}

func (m GaugeVec) Collect(chan<- prometheus.Metric) {
	panic("implement me")
}

func (m GaugeVec) WithLabelValues(lvs ...string) prometheus.Gauge {
	panic("implement me")
}
