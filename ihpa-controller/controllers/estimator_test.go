package controllers

import (
	"reflect"
	"strings"
	"testing"

	"github.com/cyberagent-oss/intelligent-hpa/ihpa-controller/controllers/metricprovider/datadog"
)

func TestAdjustYHat(t *testing.T) {
	tests := []struct {
		prevEd     *EstimateDatum
		currEd     *EstimateDatum
		prevActual float64
		expected   float64
	}{
		{
			// normal case
			prevEd: &EstimateDatum{
				EstimateUnixTime: 0,
				UnixTime:         10,
				YHat:             4.0,
				UpperYHat:        10.0,
				LowerYHat:        1.0,
			},
			currEd: &EstimateDatum{
				EstimateUnixTime: 10,
				UnixTime:         15,
				YHat:             8.0,
				UpperYHat:        20.0,
				LowerYHat:        2.0,
			},
			prevActual: 7.0,
			expected:   14.0,
		},
		{
			// big scale case
			prevEd: &EstimateDatum{
				EstimateUnixTime: 0,
				YHat:             400.0,
				UpperYHat:        1000.0,
				LowerYHat:        100.0,
			},
			currEd: &EstimateDatum{
				EstimateUnixTime: 10,
				YHat:             8.0,
				UpperYHat:        20.0,
				LowerYHat:        2.0,
			},
			prevActual: 700.0,
			expected:   14.0,
		},
		{
			// max edge case
			prevEd: &EstimateDatum{
				EstimateUnixTime: 0,
				YHat:             4.0,
				UpperYHat:        10.0,
				LowerYHat:        1.0,
			},
			currEd: &EstimateDatum{
				EstimateUnixTime: 10,
				YHat:             8.0,
				UpperYHat:        20.0,
				LowerYHat:        2.0,
			},
			prevActual: 10.0,
			expected:   20.0,
		},
		{
			// max case
			prevEd: &EstimateDatum{
				EstimateUnixTime: 0,
				YHat:             4.0,
				UpperYHat:        10.0,
				LowerYHat:        1.0,
			},
			currEd: &EstimateDatum{
				EstimateUnixTime: 10,
				YHat:             8.0,
				UpperYHat:        20.0,
				LowerYHat:        2.0,
			},
			prevActual: 30.0,
			expected:   20.0,
		},
		{
			// min edge case
			prevEd: &EstimateDatum{
				EstimateUnixTime: 0,
				YHat:             4.0,
				UpperYHat:        10.0,
				LowerYHat:        1.0,
			},
			currEd: &EstimateDatum{
				EstimateUnixTime: 10,
				YHat:             8.0,
				UpperYHat:        20.0,
				LowerYHat:        2.0,
			},
			prevActual: 1.0,
			expected:   2.0,
		},
		{
			// min case
			prevEd: &EstimateDatum{
				EstimateUnixTime: 0,
				YHat:             4.0,
				UpperYHat:        10.0,
				LowerYHat:        1.0,
			},
			currEd: &EstimateDatum{
				EstimateUnixTime: 10,
				YHat:             8.0,
				UpperYHat:        20.0,
				LowerYHat:        2.0,
			},
			prevActual: 0.5,
			expected:   2.0,
		},
		{
			// minus lower yhat
			prevEd: &EstimateDatum{
				EstimateUnixTime: 0,
				YHat:             1.0,
				UpperYHat:        7.0,
				LowerYHat:        -2.0,
			},
			currEd: &EstimateDatum{
				EstimateUnixTime: 10,
				YHat:             8.0,
				UpperYHat:        20.0,
				LowerYHat:        2.0,
			},
			prevActual: 4.0,
			expected:   14.0,
		},
		{
			// minus lower yhat and yhat
			prevEd: &EstimateDatum{
				EstimateUnixTime: 0,
				YHat:             -2.0,
				UpperYHat:        4.0,
				LowerYHat:        -5.0,
			},
			currEd: &EstimateDatum{
				EstimateUnixTime: 10,
				YHat:             8.0,
				UpperYHat:        20.0,
				LowerYHat:        2.0,
			},
			prevActual: 1.0,
			expected:   14.0,
		},
		{
			// minus all yhat
			prevEd: &EstimateDatum{
				EstimateUnixTime: 0,
				YHat:             -7.0,
				UpperYHat:        -1.0,
				LowerYHat:        -10.0,
			},
			currEd: &EstimateDatum{
				EstimateUnixTime: 10,
				YHat:             8.0,
				UpperYHat:        20.0,
				LowerYHat:        2.0,
			},
			prevActual: -4.0,
			expected:   14.0,
		},
		{
			// illegal data case
			prevEd: &EstimateDatum{
				EstimateUnixTime: 0,
				YHat:             0.0,
				UpperYHat:        -200.0,
				LowerYHat:        100.0,
			},
			currEd: &EstimateDatum{
				EstimateUnixTime: 10,
				YHat:             8.0,
				UpperYHat:        20.0,
				LowerYHat:        2.0,
			},
			prevActual: 30.0,
			expected:   8.0,
		},
	}

	for _, tt := range tests {
		got := tt.currEd.adjustYHat(tt.prevEd, tt.prevActual)
		if got != tt.expected {
			t.Fatalf("adjusted yhat is not match (got=%.3f, exp=%.3f, prevEd=%#v, currEd=%#v, prevActual=%.3f)", got, tt.expected, tt.prevEd, tt.currEd, tt.prevActual)
		}
	}
}

func TestUpdateEstimateTarget(t *testing.T) {
	dummyByteCh1 := make(chan []byte)
	dummyByteCh2 := make(chan []byte)

	dummyStructCh1 := make(chan struct{})
	dummyStructCh2 := make(chan struct{})

	dummyProvider1 := &datadog.Datadog{
		APIKey: "xxx",
		APPKey: "yyy",
	}
	dummyProvider2 := &datadog.Datadog{
		APIKey: "aaa",
		APPKey: "bbb",
	}

	tests := []struct {
		base     EstimateTarget
		patch    EstimateTarget
		expected EstimateTarget
		hasError bool
	}{
		{
			base: EstimateTarget{
				ID:              "a",
				EstimateMode:    "raw",
				GapMinutes:      10,
				MetricName:      "metric1",
				MetricTags:      []string{"hello", "world"},
				BaseMetricName:  "base-metric1",
				BaseMetricTags:  []string{"hello", "world", "foo"},
				MetricProvider:  dummyProvider1,
				DataCh:          dummyByteCh1,
				estimatorStopCh: dummyStructCh1,
			},
			patch: EstimateTarget{
				ID:             "a",
				EstimateMode:   "adjust",
				GapMinutes:     5,
				MetricName:     "metric2",
				MetricTags:     []string{"hello"},
				BaseMetricName: "base-metric2",
				BaseMetricTags: []string{"hello", "foo"},
				MetricProvider: dummyProvider2,
			},
			expected: EstimateTarget{
				ID:              "a",
				EstimateMode:    "adjust",
				GapMinutes:      5,
				MetricName:      "metric2",
				MetricTags:      []string{"hello"},
				BaseMetricName:  "base-metric2",
				BaseMetricTags:  []string{"hello", "foo"},
				MetricProvider:  dummyProvider2,
				DataCh:          dummyByteCh1,
				estimatorStopCh: dummyStructCh1,
			},
			hasError: false,
		},
		{
			base: EstimateTarget{
				ID:              "a",
				EstimateMode:    "raw",
				GapMinutes:      10,
				MetricName:      "metric1",
				MetricTags:      []string{"hello", "world"},
				BaseMetricName:  "base-metric1",
				BaseMetricTags:  []string{"hello", "world", "foo"},
				MetricProvider:  dummyProvider1,
				DataCh:          dummyByteCh1,
				estimatorStopCh: dummyStructCh1,
			},
			patch: EstimateTarget{
				ID:             "a",
				EstimateMode:   "adjust",
				BaseMetricTags: []string{"hello", "foo"},
			},
			expected: EstimateTarget{
				ID:              "a",
				EstimateMode:    "adjust",
				GapMinutes:      10,
				MetricName:      "metric1",
				MetricTags:      []string{"hello", "world"},
				BaseMetricName:  "base-metric1",
				BaseMetricTags:  []string{"hello", "foo"},
				MetricProvider:  dummyProvider1,
				DataCh:          dummyByteCh1,
				estimatorStopCh: dummyStructCh1,
			},
			hasError: false,
		},
		{
			base: EstimateTarget{
				ID:              "a",
				EstimateMode:    "raw",
				GapMinutes:      10,
				MetricName:      "metric1",
				MetricTags:      []string{"hello", "world"},
				BaseMetricName:  "base-metric1",
				BaseMetricTags:  []string{"hello", "world", "foo"},
				MetricProvider:  dummyProvider1,
				DataCh:          dummyByteCh1,
				estimatorStopCh: dummyStructCh1,
			},
			patch: EstimateTarget{
				ID:              "a",
				DataCh:          dummyByteCh2,
				estimatorStopCh: dummyStructCh2,
			},
			expected: EstimateTarget{
				ID:              "a",
				EstimateMode:    "raw",
				GapMinutes:      10,
				MetricName:      "metric1",
				MetricTags:      []string{"hello", "world"},
				BaseMetricName:  "base-metric1",
				BaseMetricTags:  []string{"hello", "world", "foo"},
				MetricProvider:  dummyProvider1,
				DataCh:          dummyByteCh1,
				estimatorStopCh: dummyStructCh1,
			},
			hasError: false,
		},
		{
			base: EstimateTarget{
				ID:              "a",
				EstimateMode:    "raw",
				GapMinutes:      10,
				MetricName:      "metric1",
				MetricTags:      []string{"hello", "world"},
				BaseMetricName:  "base-metric1",
				BaseMetricTags:  []string{"hello", "world", "foo"},
				MetricProvider:  dummyProvider1,
				DataCh:          dummyByteCh1,
				estimatorStopCh: dummyStructCh1,
			},
			patch: EstimateTarget{
				ID:           "b",
				EstimateMode: "adjust",
			},
			expected: EstimateTarget{},
			hasError: true,
		},
	}

	for _, tt := range tests {
		if err := tt.base.updateEstimateTarget(&tt.patch); err != nil {
			if tt.hasError {
				continue
			} else {
				t.Fatal(err)
			}
		} else {
			if tt.hasError {
				t.Fatalf("this case must be error")
			}
		}

		if !reflect.DeepEqual(tt.base, tt.expected) {
			t.Fatalf("updated EstimateTarget is not match (got=%#v, expected=%#v)", tt.base, tt.expected)
		}
	}
}

func TestReadEstimateDataAsCSV(t *testing.T) {
	tests := []struct {
		input    string
		expected []EstimateDatum
		hasError bool
	}{
		{
			input: `timestamp,yhat,yhat_upper,yhat_lower
100,10.0,11.0,9.0
101,10.5,11.5,9.5
102,11.0,12.0,10.0`,
			expected: []EstimateDatum{
				{
					UnixTime:  100,
					YHat:      10.0,
					UpperYHat: 11.0,
					LowerYHat: 9.0,
				},
				{
					UnixTime:  101,
					YHat:      10.5,
					UpperYHat: 11.5,
					LowerYHat: 9.5,
				},
				{
					UnixTime:  102,
					YHat:      11.0,
					UpperYHat: 12.0,
					LowerYHat: 10.0,
				},
			},
			hasError: false,
		},
		{
			input: `timestamp,yhat,yhat_upper,yhat_lower
100,10.0,11.0,9.0`,
			expected: []EstimateDatum{
				{
					UnixTime:  100,
					YHat:      10.0,
					UpperYHat: 11.0,
					LowerYHat: 9.0,
				},
			},
			hasError: false,
		},
		{
			input: `yhat_lower,yhat,timestamp,yhat_upper
9.0,10.0,100,11.0`,
			expected: []EstimateDatum{
				{
					UnixTime:  100,
					YHat:      10.0,
					UpperYHat: 11.0,
					LowerYHat: 9.0,
				},
			},
			hasError: false,
		},
		{
			input: `timestamp,yhat,yhat_upper,yhat_lower
invalid,invalid,invalid,invalid`,
			expected: []EstimateDatum{},
			hasError: false,
		},
		{
			input: `timestamp,yhat,yhat_upper
100,10.0,11.0`,
			expected: nil,
			hasError: true,
		},
		{
			input:    `timestamp,yhat,yhat_upper,yhat_lower`,
			expected: nil,
			hasError: true,
		},
		{
			input:    ``,
			expected: nil,
			hasError: true,
		},
	}

	for _, tt := range tests {
		r := strings.NewReader(tt.input)
		got, err := readEstimateDataAsCSV(r)
		if tt.hasError {
			if err == nil {
				t.Fatalf("this case must have error")
			}
			continue
		} else {
			if err != nil {
				t.Fatal(err)
			}
		}

		if !reflect.DeepEqual(got, tt.expected) {
			t.Fatalf("data is not match (got=%v, exp=%v)", got, tt.expected)
		}
	}
}

func TestJoinEstimateData(t *testing.T) {
	tests := []struct {
		newEds   []EstimateDatum
		oldEds   []EstimateDatum
		expected []EstimateDatum
	}{
		{
			newEds: []EstimateDatum{
				{
					EstimateUnixTime: 100,
					YHat:             10.0,
					UpperYHat:        11.0,
					LowerYHat:        9.0,
				},
			},
			oldEds: []EstimateDatum{
				{
					EstimateUnixTime: 50,
					YHat:             10.0,
					UpperYHat:        11.0,
					LowerYHat:        9.0,
				},
			},
			expected: []EstimateDatum{
				{
					EstimateUnixTime: 50,
					YHat:             10.0,
					UpperYHat:        11.0,
					LowerYHat:        9.0,
				},
				{
					EstimateUnixTime: 100,
					YHat:             10.0,
					UpperYHat:        11.0,
					LowerYHat:        9.0,
				},
			},
		},
		{
			newEds: []EstimateDatum{
				{
					EstimateUnixTime: 100,
					YHat:             10.0,
					UpperYHat:        11.0,
					LowerYHat:        9.0,
				},
				{
					EstimateUnixTime: 101,
					YHat:             10.5,
					UpperYHat:        11.5,
					LowerYHat:        9.5,
				},
			},
			oldEds: []EstimateDatum{
				{
					EstimateUnixTime: 99,
					YHat:             20.0,
					UpperYHat:        21.0,
					LowerYHat:        19.0,
				},
				{
					EstimateUnixTime: 100,
					YHat:             20.5,
					UpperYHat:        21.5,
					LowerYHat:        19.5,
				},
			},
			expected: []EstimateDatum{
				{
					EstimateUnixTime: 99,
					YHat:             20.0,
					UpperYHat:        21.0,
					LowerYHat:        19.0,
				},
				{
					EstimateUnixTime: 100,
					YHat:             10.0,
					UpperYHat:        11.0,
					LowerYHat:        9.0,
				},
				{
					EstimateUnixTime: 101,
					YHat:             10.5,
					UpperYHat:        11.5,
					LowerYHat:        9.5,
				},
			},
		},
		{
			newEds: []EstimateDatum{},
			oldEds: []EstimateDatum{
				{
					EstimateUnixTime: 50,
					YHat:             10.0,
					UpperYHat:        11.0,
					LowerYHat:        9.0,
				},
			},
			expected: []EstimateDatum{
				{
					EstimateUnixTime: 50,
					YHat:             10.0,
					UpperYHat:        11.0,
					LowerYHat:        9.0,
				},
			},
		},
		{
			newEds: []EstimateDatum{
				{
					EstimateUnixTime: 100,
					YHat:             10.0,
					UpperYHat:        11.0,
					LowerYHat:        9.0,
				},
			},
			oldEds: []EstimateDatum{},
			expected: []EstimateDatum{
				{
					EstimateUnixTime: 100,
					YHat:             10.0,
					UpperYHat:        11.0,
					LowerYHat:        9.0,
				},
			},
		},
		{
			newEds:   []EstimateDatum{},
			oldEds:   []EstimateDatum{},
			expected: []EstimateDatum{},
		},
	}

	for _, tt := range tests {
		got := joinEstimateData(tt.newEds, tt.oldEds)

		if !reflect.DeepEqual(got, tt.expected) {
			t.Fatalf("data is not match (got=%v, exp=%v)", got, tt.expected)
		}
	}
}

func TestPastEstimateDatumQueueEnqueue(t *testing.T) {
	tests := []struct {
		q    PastEstimateDatumQueue
		d    *EstimateDatum
		expq PastEstimateDatumQueue
	}{
		{
			q:    PastEstimateDatumQueue{{UnixTime: 0}, {UnixTime: 10}},
			d:    &EstimateDatum{UnixTime: 20},
			expq: PastEstimateDatumQueue{{UnixTime: 0}, {UnixTime: 10}, {UnixTime: 20}},
		},
		{
			q:    PastEstimateDatumQueue{},
			d:    &EstimateDatum{UnixTime: 20},
			expq: PastEstimateDatumQueue{{UnixTime: 20}},
		},
		{
			q:    PastEstimateDatumQueue{{UnixTime: 0}, {UnixTime: 10}},
			d:    nil,
			expq: PastEstimateDatumQueue{{UnixTime: 0}, {UnixTime: 10}},
		},
	}

	for _, tt := range tests {
		tt.q.enqueue(tt.d)
		if !reflect.DeepEqual(tt.q, tt.expq) {
			t.Fatalf("queue is not match (got=%#v, exp=%#v)", tt.q, tt.expq)
		}
	}
}

func TestPastEstimateDatumQueueDequeue(t *testing.T) {
	tests := []struct {
		q    PastEstimateDatumQueue
		expd *EstimateDatum
		expq PastEstimateDatumQueue
	}{
		{
			q:    PastEstimateDatumQueue{{UnixTime: 0}, {UnixTime: 10}},
			expd: &EstimateDatum{UnixTime: 0},
			expq: PastEstimateDatumQueue{{UnixTime: 10}},
		},
		{
			q:    PastEstimateDatumQueue{},
			expd: nil,
			expq: PastEstimateDatumQueue{},
		},
	}

	for _, tt := range tests {
		got := tt.q.dequeue()
		if !reflect.DeepEqual(got, tt.expd) {
			t.Fatalf("data is not match (got=%#v, exp=%#v)", got, tt.expd)
		}
		if !reflect.DeepEqual(tt.q, tt.expq) {
			t.Fatalf("queue is not match (got=%#v, exp=%#v)", tt.q, tt.expq)
		}
	}
}

func TestPastEstimateDatumQueuePeek(t *testing.T) {
	tests := []struct {
		q    PastEstimateDatumQueue
		expd *EstimateDatum
		expq PastEstimateDatumQueue
	}{
		{
			q:    PastEstimateDatumQueue{{UnixTime: 0}, {UnixTime: 10}},
			expd: &EstimateDatum{UnixTime: 0},
			expq: PastEstimateDatumQueue{{UnixTime: 0}, {UnixTime: 10}},
		},
		{
			q:    PastEstimateDatumQueue{},
			expd: nil,
			expq: PastEstimateDatumQueue{},
		},
	}

	for _, tt := range tests {
		got := tt.q.peek()
		if !reflect.DeepEqual(got, tt.expd) {
			t.Fatalf("data is not match (got=%#v, exp=%#v)", got, tt.expd)
		}
		if !reflect.DeepEqual(tt.q, tt.expq) {
			t.Fatalf("queue is not match (got=%#v, exp=%#v)", tt.q, tt.expq)
		}
	}
}

func TestPastEstimateDatumQueueSeekByUnixTime(t *testing.T) {
	tests := []struct {
		q        PastEstimateDatumQueue
		seekTime int64
		expd     *EstimateDatum
		expq     PastEstimateDatumQueue
	}{
		{
			q:        PastEstimateDatumQueue{{UnixTime: 0}, {UnixTime: 10}, {UnixTime: 20}},
			seekTime: 15,
			expd:     &EstimateDatum{UnixTime: 10},
			expq:     PastEstimateDatumQueue{{UnixTime: 20}},
		},
		{
			q:        PastEstimateDatumQueue{{UnixTime: 0}, {UnixTime: 10}, {UnixTime: 20}},
			seekTime: 20,
			expd:     &EstimateDatum{UnixTime: 20},
			expq:     PastEstimateDatumQueue{},
		},
		{
			q:        PastEstimateDatumQueue{{UnixTime: 0}, {UnixTime: 10}, {UnixTime: 20}},
			seekTime: 25,
			expd:     &EstimateDatum{UnixTime: 20},
			expq:     PastEstimateDatumQueue{},
		},
		{
			q:        PastEstimateDatumQueue{{UnixTime: 10}, {UnixTime: 20}},
			seekTime: 5,
			expd:     nil,
			expq:     PastEstimateDatumQueue{{UnixTime: 10}, {UnixTime: 20}},
		},
		{
			q:        PastEstimateDatumQueue{},
			seekTime: 5,
			expd:     nil,
			expq:     PastEstimateDatumQueue{},
		},
	}

	for _, tt := range tests {
		got := tt.q.seekByUnixTime(tt.seekTime)
		if !reflect.DeepEqual(got, tt.expd) {
			t.Fatalf("data is not match (got=%#v, exp=%#v)", got, tt.expd)
		}
		if !reflect.DeepEqual(tt.q, tt.expq) {
			t.Fatalf("queue is not match (got=%#v, exp=%#v)", tt.q, tt.expq)
		}
	}
}
