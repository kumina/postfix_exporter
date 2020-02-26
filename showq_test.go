package main

import (
	"github.com/kumina/postfix_exporter/mock"
	"github.com/stretchr/testify/assert"
	"os"
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
		expectedTotalCount float64
	}{
		{
			name: "basic test",
			args: args{
				file: "testdata/showq.txt",
			},
			wantErr:            false,
			expectedTotalCount: 118702,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, err := os.Open(tt.args.file)
			if err != nil {
				t.Error(err)
			}

			sizeHistogram := mock.NewHistogramVecMock()
			ageHistogram := mock.NewHistogramVecMock()
			if err := CollectTextualShowqFromScanner(sizeHistogram, ageHistogram, file); (err != nil) != tt.wantErr {
				t.Errorf("CollectShowqFromReader() error = %v, wantErr %v", err, tt.wantErr)
			}
			assert.Equal(t, tt.expectedTotalCount, sizeHistogram.GetSum(), "Expected a lot more data.")
			assert.Less(t, 0.0, ageHistogram.GetSum(), "Age not greater than 0")
		})
	}
}
