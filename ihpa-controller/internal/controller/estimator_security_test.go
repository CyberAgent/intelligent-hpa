package controller

import (
	"strings"
	"testing"
	"time"

	"github.com/go-logr/logr"
)

func TestRejectInvalidForecastRows(t *testing.T) {
	for _, row := range []string{"bad,1,2,0", "1,NaN,2,0", "1,1,+Inf,0", "1,1,2,-Inf", "-1,1,2,0", "9223372036854775807,1,2,0"} {
		t.Run(row, func(t *testing.T) {
			if _, err := readEstimateDataAsCSV(strings.NewReader("timestamp,yhat,yhat_upper,yhat_lower\n" + row + "\n")); err == nil {
				t.Fatal("invalid batch accepted")
			}
		})
	}
}

func TestExpiredForecasts(t *testing.T) {
	for _, data := range [][]EstimateDatum{nil, {{EstimateUnixTime: 9}}, {{EstimateUnixTime: 10}}} {
		if got := futureEstimateData(data, 10); len(got) != 0 {
			t.Fatalf("expired batch retained: %v", got)
		}
	}
	data := []EstimateDatum{{EstimateUnixTime: 9}, {EstimateUnixTime: 11}}
	if got := futureEstimateData(data, 10); len(got) != 1 || got[0].EstimateUnixTime != 11 {
		t.Fatalf("future forecast lost: %v", got)
	}
}

func TestEstimatorSurvivesUnusableForecasts(t *testing.T) {
	for name, batch := range map[string]string{
		"invalid": "timestamp,yhat,yhat_upper,yhat_lower\nbad,1,2,0\n",
		"expired": "timestamp,yhat,yhat_upper,yhat_lower\n1,1,2,0\n",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			input := make(chan []byte, 1)
			input <- []byte(batch)
			stop := make(chan struct{})
			done := make(chan struct{})
			target := EstimateTarget{DataCh: input, estimatorStopCh: stop, Logger: logr.Discard()}
			go func() { defer close(done); target.estimator() }()
			time.Sleep(5200 * time.Millisecond)
			select {
			case <-done:
				t.Fatal("estimator exited before receiving a stop request")
			default:
			}
			close(stop)
			select {
			case <-done:
			case <-time.After(time.Second):
				t.Fatal("estimator did not stop after unusable forecast")
			}
		})
	}
}
