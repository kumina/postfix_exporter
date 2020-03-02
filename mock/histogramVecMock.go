package mock

import "github.com/prometheus/client_golang/prometheus"

type HistogramVecMock struct {
	mock HistogramMock
}

func (m *HistogramVecMock) Describe(chan<- *prometheus.Desc) {
	panic("implement me")
}

func (m *HistogramVecMock) GetMetricWith(prometheus.Labels) (prometheus.Observer, error) {
	panic("implement me")
}

func (m *HistogramVecMock) GetMetricWithLabelValues(lvs ...string) (prometheus.Observer, error) {
	panic("implement me")
}

func (m *HistogramVecMock) With(prometheus.Labels) prometheus.Observer {
	panic("implement me")
}

func (m *HistogramVecMock) WithLabelValues(...string) prometheus.Observer {
	return m.mock
}

func (m *HistogramVecMock) CurryWith(prometheus.Labels) (prometheus.ObserverVec, error) {
	panic("implement me")
}

func (m *HistogramVecMock) MustCurryWith(prometheus.Labels) prometheus.ObserverVec {
	panic("implement me")
}

func (m *HistogramVecMock) Collect(chan<- prometheus.Metric) {
	panic("implement me")
}
func (m *HistogramVecMock) GetSum() float64 {
	return *m.mock.sum
}

func NewHistogramVecMock() *HistogramVecMock {
	return &HistogramVecMock{mock: *NewHistogramMock()}
}
