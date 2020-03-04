package showq

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"github.com/kumina/postfix_exporter/mock"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path"
	"testing"
)

func TestCollectShowqFromReader(t *testing.T) {
	type args struct {
		file string
	}
	tests := []struct {
		name               string
		args               args
		wantErr            bool
		expectedTotalSize  float64
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

			sizeHistogram := mock.NewHistogramVecMock()
			ageHistogram := mock.NewHistogramVecMock()
			showQ := NewShowQCollector(tt.args.file, mock.GaugeVec{}, "10s")
			showQ.ageHistogram = ageHistogram
			showQ.sizeHistogram = sizeHistogram
			if err := showQ.CollectShowqFromSocket(socket); (err != nil) != tt.wantErr {
				t.Errorf("CollectShowqFromSocket() error = %v, wantErr %v", err, tt.wantErr)
			}
			assert.Equal(t, tt.expectedTotalSize, sizeHistogram.GetSum(), "Expected a lot more data.")
			assert.Equal(t, tt.expectedTotalCount, sizeHistogram.GetCount(), "Wrong number of points counted.")
			assert.Less(t, 0.0, ageHistogram.GetSum(), "Age not greater than 0")
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
