package controller

import (
	"strings"
	"testing"
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
