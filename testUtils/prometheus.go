package testUtils

import (
	"fmt"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	prometheusClient "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
)

func GetCount(counter prometheus.Collector) (int, []string, error) {
	var count = 0
	metricsChan := make(chan prometheus.Metric)
	go func() {
		counter.Collect(metricsChan)
		close(metricsChan)
	}()
	for metric := range metricsChan {
		metricDto := prometheusClient.Metric{}
		if err := metric.Write(&metricDto); err != nil {
			return 0, nil, fmt.Errorf("failed to write the metric: %v", err)
		}
		switch {
		case metricDto.Gauge != nil:
			count += int(*metricDto.Gauge.Value)
		case metricDto.Counter != nil:
			count += int(*metricDto.Counter.Value)
		case metricDto.Summary != nil:
			count += int(*metricDto.Summary.SampleCount)
		case metricDto.Histogram != nil:
			count += int(*metricDto.Histogram.SampleCount)
		}
	}
	descChan := make(chan *prometheus.Desc)
	go func() {
		counter.Describe(descChan)
		close(descChan)
	}()
	var descriptions []string
	for desc := range descChan {
		descriptions = append(descriptions, desc.String())
	}
	return count, descriptions, nil
}

func AssertCounterEquals(t *testing.T, counter prometheus.Collector, expected int, message string) {
	if counter != nil {
		count, _, err := GetCount(counter)
		if err != nil {
			t.Errorf("could not get count: %v", err)
		}
		assert.Equal(t, expected, count, message)
	}
}

func AssertSumEquals(t *testing.T, collector prometheus.Collector, expected int, message string) {
	var sum = 0
	sum, err := GetSum(collector, sum)
	if err != nil {
		t.Fatalf("Failed to get Sum: %v", err)
	}
	assert.Equal(t, expected, sum, message)
}

func AssertSumLessThan(t *testing.T, collector prometheus.Collector, expected int, message string) {
	var sum = 0
	sum, err := GetSum(collector, sum)
	if err != nil {
		t.Fatalf("Failed to get Sum: %v", err)
	}
	assert.Less(t, expected, sum, message)
}

func GetSum(collector prometheus.Collector, sum int) (int, error) {
	if collector != nil {
		metricsChan := make(chan prometheus.Metric)
		go func() {
			collector.Collect(metricsChan)
			close(metricsChan)
		}()
		for metric := range metricsChan {
			metricDto := prometheusClient.Metric{}
			err := metric.Write(&metricDto)
			if err != nil {
				return 0, fmt.Errorf("cannot write metric: %v", err)
			}
			switch {
			case metricDto.Histogram != nil:
				sum += int(*metricDto.Histogram.SampleSum)
			case metricDto.Summary != nil:
				sum += int(*metricDto.Summary.SampleSum)
			default:
				return 0, fmt.Errorf("type not implemented: %T", collector)
			}
		}
	}
	return sum, nil
}
