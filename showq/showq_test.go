package showq

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path"
	"testing"

	"github.com/kumina/postfix_exporter/mock"
	"github.com/prometheus/client_golang/prometheus"
	io_prometheus_client "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
)

func TestCollectShowqFromReader(t *testing.T) {
	type args struct {
		file string
	}
	tests := []struct {
		name               string
		args               args
		wantErr            bool
		expectedTotalSize  int
		expectedTotalCount int
	}{
		{
			name: "basic test",
			args: args{
				file: "testdata/showq.txt",
			},
			wantErr:            false,
			expectedTotalCount: 25,
			expectedTotalSize:  122790,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancelFunc := context.WithCancel(context.Background())

			socket, ch, err := writeToSocket(ctx, tt.args.file)
			if err != nil {
				t.Errorf("failed to write to socket: %v", err)
			}
			defer cancelFunc()

			sizeHistogram := prometheus.NewHistogramVec(prometheus.HistogramOpts{}, []string{"active"})
			ageHistogram := prometheus.NewHistogramVec(prometheus.HistogramOpts{}, []string{"active"})
			showQ := NewShowQCollector(tt.args.file, mock.GaugeVec{}, "10s")
			showQ.ageHistogram = ageHistogram
			showQ.sizeHistogram = sizeHistogram
			if err := showQ.CollectShowqFromSocket(socket); (err != nil) != tt.wantErr {
				t.Errorf("CollectShowqFromSocket() error = %v, wantErr %v", err, tt.wantErr)
			}
			assertSumEquals(t, sizeHistogram, tt.expectedTotalSize, "Expected a lot more data")
			assertCounterEquals(t, sizeHistogram, tt.expectedTotalCount, "Wrong number of points counted.")
			assertSumLessThan(t, ageHistogram, 0, "Age not greater than 0")
			cancelFunc()
			<-ch
		})
	}
}

func writeToSocket(ctx context.Context, filename string) (string, <-chan interface{}, error) {
	ch := make(chan interface{})
	dataFile, err := os.Open(filename)
	if err != nil {
		return "", nil, fmt.Errorf("failed to open file %s: %v", filename, err)
	}
	defer dataFile.Close()
	content, err := ioutil.ReadAll(dataFile)
	if err != nil {
		return "", nil, fmt.Errorf("failed to read file %s: %v", filename, err)
	}
	socketName := path.Join(os.TempDir(), "go.sock")
	var listenConfig net.ListenConfig
	lsnr, err := listenConfig.Listen(ctx, "unix", socketName)
	if err != nil {
		log.Fatal("Listen error: ", err)
	}

	go func() {
		for {
			connection, err := lsnr.Accept()
			select {
			case <-ctx.Done():
				return
			default:
			}
			if err != nil {
				log.Fatal("Accept error: ", err)
			}
			scanner := bufio.NewScanner(bytes.NewReader(content))
			for scanner.Scan() {
				line := scanner.Text()
				_, err = connection.Write([]byte(fmt.Sprintln(line)))
			}
			if err != nil {
				log.Fatal("Write error: ", err)
			}
			err = connection.Close()
			if err != nil {
				log.Printf("Failed to close the connection: %v", err)
			}
		}
	}()
	go func() {
		<-ctx.Done()
		lsnr.Close()
		log.Printf("Closed the listener")
		ch <- struct{}{}
	}()
	return socketName, ch, nil
}

func assertCounterEquals(t *testing.T, counter prometheus.Collector, expected int, message string) {

	if counter != nil {
		switch counter.(type) {
		case *prometheus.CounterVec:
			counter := counter.(*prometheus.CounterVec)
			metricsChan := make(chan prometheus.Metric)
			go func() {
				counter.Collect(metricsChan)
				close(metricsChan)
			}()
			var count int = 0
			for metric := range metricsChan {
				metricDto := io_prometheus_client.Metric{}
				metric.Write(&metricDto)
				count += int(*metricDto.Counter.Value)
			}
			assert.Equal(t, expected, count, message)
		case prometheus.Counter:
			metricsChan := make(chan prometheus.Metric)
			go func() {
				counter.Collect(metricsChan)
				close(metricsChan)
			}()
			var count int = 0
			for metric := range metricsChan {
				metricDto := io_prometheus_client.Metric{}
				metric.Write(&metricDto)
				count += int(*metricDto.Counter.Value)
			}
			assert.Equal(t, expected, count, message)
		case *prometheus.HistogramVec:
			counter := counter.(*prometheus.HistogramVec)
			metricsChan := make(chan prometheus.Metric)
			go func() {
				counter.Collect(metricsChan)
				close(metricsChan)
			}()
			var count int = 0
			for metric := range metricsChan {
				metricDto := io_prometheus_client.Metric{}
				metric.Write(&metricDto)
				count += int(*metricDto.Histogram.SampleCount)
			}
			assert.Equal(t, expected, count, message)
		case prometheus.Histogram:
			metricsChan := make(chan prometheus.Metric)
			go func() {
				counter.Collect(metricsChan)
				close(metricsChan)
			}()
			var count int = 0
			for metric := range metricsChan {
				metricDto := io_prometheus_client.Metric{}
				metric.Write(&metricDto)
				count += int(*metricDto.Histogram.SampleCount)
			}
			assert.Equal(t, expected, count, message)
		case *prometheus.GaugeVec:
			counter := counter.(*prometheus.GaugeVec)
			metricsChan := make(chan prometheus.Metric)
			go func() {
				counter.Collect(metricsChan)
				close(metricsChan)
			}()
			var count int = 0
			for metric := range metricsChan {
				metricDto := io_prometheus_client.Metric{}
				metric.Write(&metricDto)
				count += int(*metricDto.Gauge.Value)
			}
			assert.Equal(t, expected, count, message)
		case prometheus.Gauge:
			metricsChan := make(chan prometheus.Metric)
			go func() {
				counter.Collect(metricsChan)
				close(metricsChan)
			}()
			var count int = 0
			for metric := range metricsChan {
				metricDto := io_prometheus_client.Metric{}
				metric.Write(&metricDto)
				count += int(*metricDto.Gauge.Value)
			}
			assert.Equal(t, expected, count, message)
		default:
			t.Fatalf("Type not implemented: %T", counter)
		}
	}
}

func assertSumEquals(t *testing.T, counter prometheus.Collector, expected int, message string) {

	if counter != nil {
		switch counter.(type) {
		case *prometheus.HistogramVec:
			counter := counter.(*prometheus.HistogramVec)
			metricsChan := make(chan prometheus.Metric)
			go func() {
				counter.Collect(metricsChan)
				close(metricsChan)
			}()
			var count int = 0
			for metric := range metricsChan {
				metricDto := io_prometheus_client.Metric{}
				metric.Write(&metricDto)
				count += int(*metricDto.Histogram.SampleSum)
			}
			assert.Equal(t, expected, count, message)
		case prometheus.Histogram:
			metricsChan := make(chan prometheus.Metric)
			go func() {
				counter.Collect(metricsChan)
				close(metricsChan)
			}()
			var count int = 0
			for metric := range metricsChan {
				metricDto := io_prometheus_client.Metric{}
				metric.Write(&metricDto)
				count += int(*metricDto.Histogram.SampleSum)
			}
			assert.Equal(t, expected, count, message)
		default:
			t.Fatalf("Type not implemented: %T", counter)
		}
	}
}
func assertSumLessThan(t *testing.T, counter prometheus.Collector, expected int, message string) {

	if counter != nil {
		switch counter.(type) {
		case *prometheus.HistogramVec:
			counter := counter.(*prometheus.HistogramVec)
			metricsChan := make(chan prometheus.Metric)
			go func() {
				counter.Collect(metricsChan)
				close(metricsChan)
			}()
			var count int = 0
			for metric := range metricsChan {
				metricDto := io_prometheus_client.Metric{}
				metric.Write(&metricDto)
				count += int(*metricDto.Histogram.SampleSum)
			}
			assert.Less(t, expected, count, message)
		case prometheus.Histogram:
			metricsChan := make(chan prometheus.Metric)
			go func() {
				counter.Collect(metricsChan)
				close(metricsChan)
			}()
			var count int = 0
			for metric := range metricsChan {
				metricDto := io_prometheus_client.Metric{}
				metric.Write(&metricDto)
				count += int(*metricDto.Histogram.SampleSum)
			}
			assert.Less(t, expected, count, message)
		default:
			t.Fatalf("Type not implemented: %T", counter)
		}
	}
}
