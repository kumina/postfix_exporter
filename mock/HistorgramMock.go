package mock

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_model/go"
)

type HistogramMock struct {
	sum *float64
}

func NewHistogramMock() *HistogramMock {
	return &HistogramMock{sum: new(float64)}
}

func (HistogramMock) Desc() *prometheus.Desc {
	panic("implement me")
}

func (HistogramMock) Write(*io_prometheus_client.Metric) error {
	panic("implement me")
}

func (HistogramMock) Describe(chan<- *prometheus.Desc) {
	panic("implement me")
}

func (HistogramMock) Collect(chan<- prometheus.Metric) {
	panic("implement me")
}

func (h HistogramMock) Observe(value float64) {
	*h.sum += value
}
