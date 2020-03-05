package mock

import (
	"github.com/prometheus/client_golang/prometheus"
	io_prometheus_client "github.com/prometheus/client_model/go"
)

type HistogramMock struct {
	sum   float64
	count int
}

func NewHistogramMock() *HistogramMock {
	return &HistogramMock{}
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

func (h *HistogramMock) Observe(value float64) {
	h.sum += value
	h.count++
}
