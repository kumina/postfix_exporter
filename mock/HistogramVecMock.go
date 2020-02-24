package mock

import "github.com/prometheus/client_golang/prometheus"

type HistorgramVecMock struct {
	mock HistogramMock
}

func (m *HistorgramVecMock) Describe(chan<- *prometheus.Desc) {
	panic("implement me")
}

func (m *HistorgramVecMock) GetMetricWith(prometheus.Labels) (prometheus.Observer, error) {
	panic("implement me")
}

func (m *HistorgramVecMock) GetMetricWithLabelValues(lvs ...string) (prometheus.Observer, error) {
	panic("implement me")
}

func (m *HistorgramVecMock) With(prometheus.Labels) prometheus.Observer {
	panic("implement me")
}

func (m *HistorgramVecMock) WithLabelValues(...string) prometheus.Observer {
	return m.mock
}

func (m *HistorgramVecMock) CurryWith(prometheus.Labels) (prometheus.ObserverVec, error) {
	panic("implement me")
}

func (m *HistorgramVecMock) MustCurryWith(prometheus.Labels) prometheus.ObserverVec {
	panic("implement me")
}

func (m *HistorgramVecMock) Collect(chan<- prometheus.Metric) {
	panic("implement me")
}
func (m *HistorgramVecMock) GetSum() float64 {
	return *m.mock.sum
}

func NewHistogramVecMock() *HistorgramVecMock {
	return &HistorgramVecMock{mock: *NewHistogramMock()}
}
