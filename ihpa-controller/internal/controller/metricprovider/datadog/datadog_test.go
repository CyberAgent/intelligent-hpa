package datadog

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"
	"time"
)

func TestDatadogSeriesMarshalJSON(t *testing.T) {
	tests := []struct {
		input    timeseries
		expected string
	}{
		{
			input: timeseries{
				TimeseriesItems: []timeseriesItem{
					{
						Metric: "test",
						Points: []datapoint{
							{
								unixtime: 100,
								point:    10.0,
							},
						},
						Tags: []string{"tag1", "tag2"},
					},
				},
			},
			expected: `{"series":[{"metric":"test","points":[[100,10]],"tags":["tag1","tag2"]}]}`,
		},
	}

	for _, tt := range tests {
		got, err := json.Marshal(&tt.input)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != tt.expected {
			t.Fatalf("marshal json is not match (got=%s, exp=%s)", got, tt.expected)
		}
	}
}

func TestDatadogMarshalJSON(t *testing.T) {
	tests := []struct {
		input    timeseriesItem
		expected string
	}{
		{
			input: timeseriesItem{
				Metric: "test",
				Points: []datapoint{
					{
						unixtime: 100,
						point:    10.0,
					},
				},
				Tags: []string{"tag1", "tag2"},
			},
			expected: `{"metric":"test","points":[[100,10]],"tags":["tag1","tag2"]}`,
		},
		{
			input: timeseriesItem{
				Metric: "test",
				Points: []datapoint{
					{
						unixtime: 100,
						point:    10.0,
					},
					{
						unixtime: 101,
						point:    11.5,
					},
				},
				Tags: []string{"tag1", "tag2"},
			},
			expected: `{"metric":"test","points":[[100,10],[101,11.5]],"tags":["tag1","tag2"]}`,
		},
	}

	for _, tt := range tests {
		got, err := json.Marshal(&tt.input)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != tt.expected {
			t.Fatalf("marshal json is not match (got=%s, exp=%s)", got, tt.expected)
		}
	}
}

func TestDatadogRequest(t *testing.T) {
	contents := []struct {
		path       string
		bodyPath   string
		method     string
		statusCode int
	}{
		{
			path:       "/api/v1/series",
			bodyPath:   "testdata/series/ok.json",
			method:     http.MethodPost,
			statusCode: http.StatusAccepted,
		},
		{
			path:       "/api/v1/query",
			bodyPath:   "testdata/query/kubernetes_cpu_usage_total.json",
			method:     http.MethodGet,
			statusCode: http.StatusOK,
		},
		{
			path:       "/api/v1/metrics",
			bodyPath:   "testdata/metrics/ok.json",
			method:     http.MethodPut,
			statusCode: http.StatusOK,
		},
		{
			path:       "/api/v1/metrics/none",
			bodyPath:   "testdata/metrics/none.json",
			method:     http.MethodGet,
			statusCode: http.StatusOK,
		},
		{
			path:       "/api/v1/metrics/kubernetes.cpu.usage.total",
			bodyPath:   "testdata/metrics/nanocore.json",
			method:     http.MethodGet,
			statusCode: http.StatusOK,
		},
	}

	mux := http.NewServeMux()
	for _, content := range contents {
		c := content
		mux.HandleFunc(c.path, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("Content-Type", "application/json")

			w.WriteHeader(c.statusCode)
			body, err := os.Open(c.bodyPath)
			if err != nil {
				fmt.Fprintf(w, `{"error": "file open error: %s"}`, c.bodyPath)
				return
			}

			io.Copy(w, body)
		})
	}
	server := httptest.NewServer(mux)

	testDatadogUnit(server.URL, t)
	testDatadogFetch(server.URL, t)
	testDatadogSend(server.URL, t)
}

func testDatadogUnit(url string, t *testing.T) {
	d := &Datadog{APIKey: "xxx", APPKey: "yyy"}
	mtype, munit, err := d.getUnit(url, "kubernetes.cpu.usage.total")
	if err != nil {
		t.Fatal(err)
	}
	if mtype != "gauge" || munit != "nanocore" {
		t.Fatalf("type or unit is not match (got type=%s, unit=%s, exp type=%s, unit=%s)", mtype, munit, "gauge", "nanocore")
	}

	err = d.setUnit(url, "none", mtype, munit)
	if err != nil {
		t.Fatal(err)
	}
}

func testDatadogFetch(url string, t *testing.T) {
	d := &Datadog{APIKey: "xxx", APPKey: "yyy"}
	dp, err := d.fetch(url, "kubernetes.cpu.usage.total", time.Date(2020, 3, 1, 8, 0, 0, 0, time.UTC).Unix(), []string{"mytag:test", "yourtag:test"})
	if err != nil {
		t.Fatal(err)
	}
	if dp.point != 4027508.567882628 {
		t.Fatalf("point is not match (got=%f, exp=%f)", dp.point, 4027508.567882628)
	}
}

func testDatadogSend(url string, t *testing.T) {
	d := &Datadog{APIKey: "xxx", APPKey: "yyy"}
	err := d.send(url, "none", time.Now().Unix(), 10.0, []string{"tag:test"}, "kubernetes.cpu.usage.total")
	if err != nil {
		t.Fatal(err)
	}
}

func TestBinarySearchNearTimestamp(t *testing.T) {
	tests := []struct {
		dps       []datapoint
		timestamp int64
		expected  float64
	}{
		{
			dps: []datapoint{
				{unixtime: 10, point: 10.0},
			},
			timestamp: 10,
			expected:  10.0,
		},
		{
			dps: []datapoint{
				{unixtime: 10, point: 10.0},
			},
			timestamp: 11,
			expected:  10.0,
		},
		{
			dps: []datapoint{
				{unixtime: 10, point: 10.0},
				{unixtime: 11, point: 11.0},
				{unixtime: 12, point: 12.0},
				{unixtime: 13, point: 13.0},
				{unixtime: 14, point: 14.0},
				{unixtime: 15, point: 15.0},
			},
			timestamp: 13,
			expected:  13.0,
		},
		{
			dps: []datapoint{
				{unixtime: 10, point: 10.0},
				{unixtime: 11, point: 11.0},
				{unixtime: 12, point: 12.0},
				{unixtime: 13, point: 13.0},
				{unixtime: 14, point: 14.0},
			},
			timestamp: 10,
			expected:  10.0,
		},
		{
			dps: []datapoint{
				{unixtime: 10, point: 10.0},
				{unixtime: 11, point: 11.0},
				{unixtime: 12, point: 12.0},
				{unixtime: 13, point: 13.0},
				{unixtime: 14, point: 14.0},
			},
			timestamp: 5,
			expected:  10.0,
		},
	}

	for _, tt := range tests {
		dp := binarySearchNearTimestamp(tt.dps, tt.timestamp)
		if dp.point != tt.expected {
			t.Fatalf("point is not match (got=%.1f, exp=%.1f)", dp.point, tt.expected)
		}
	}
}

func TestMergeAllSeriesDatapointToOne(t *testing.T) {
	tests := []struct {
		j        string
		expected []datapoint
	}{
		{
			j: `
{
  "series": [
    {
      "pointlist": [
        [100000, 10.0],
        [105000, 11.0],
        [110000, 12.0],
        [115000, 13.0],
        [120000, 14.0]
      ]
    },
    {
      "pointlist": [
        [100000, 20.0],
        [105000, 21.0],
        [110000, 22.0],
        [112000, 23.0],
        [114000, 24.0],
        [116000, 25.0]
      ]
    },
    {
      "pointlist": [
        [120000, 10.0],
        [115000, 10.0]
      ]
    }
  ]
}
			`,
			expected: []datapoint{
				{unixtime: 100, point: 30.0},
				{unixtime: 105, point: 32.0},
				{unixtime: 110, point: 34.0},
				{unixtime: 112, point: 23.0},
				{unixtime: 114, point: 24.0},
				{unixtime: 115, point: 23.0},
				{unixtime: 116, point: 25.0},
				{unixtime: 120, point: 24.0},
			},
		},
		{
			j: `
{
  "status": "ok",
  "res_type": "time_series",
  "from_date": 1585640700000,
  "series": [
    {
      "end": 1585641899000,
      "attributes": {},
      "metric": "nginx.net.request_per_s",
      "interval": 5,
      "tag_set": [
        "host:uruai2-ake-default-b-nydxadk"
      ],
      "start": 1585640700000,
      "length": 163,
      "query_index": 0,
      "aggr": "sum",
      "scope": "ake_cluster_name:uruai2,host:uruai2-ake-default-b-nydxadk,kube_container_name:nginx,kube_deployment:nginx,kube_namespace:loadtest",
      "pointlist": [
        [
          1585640700000,
          59.50
        ],
        [
          1585640710000,
          45.05
        ],
        [
          1585640715000,
          41.79
        ]
      ],
      "expression": "sum:nginx.net.request_per_s{ake_cluster_name:uruai2,host:uruai2-ake-default-b-nydxadk,kube_container_name:nginx,kube_deployment:nginx,kube_namespace:loadtest}",
      "unit": [
        {
          "family": "network",
          "scale_factor": 1,
          "name": "request",
          "short_name": "req",
          "plural": "requests",
          "id": 19
        },
        {
          "family": "time",
          "scale_factor": 1,
          "name": "second",
          "short_name": "s",
          "plural": "seconds",
          "id": 11
        }
      ],
      "display_name": "nginx.net.request_per_s"
    },
    {
      "end": 1585641899000,
      "attributes": {},
      "metric": "nginx.net.request_per_s",
      "interval": 5,
      "tag_set": [
        "host:uruai2-ake-default-b-nlmtind"
      ],
      "start": 1585640700000,
      "length": 163,
      "query_index": 0,
      "aggr": "sum",
      "scope": "ake_cluster_name:uruai2,host:uruai2-ake-default-b-nlmtind,kube_container_name:nginx,kube_deployment:nginx,kube_namespace:loadtest",
      "pointlist": [
        [
          1585640700000,
          44.53
        ],
        [
          1585640710000,
          51.45
        ],
        [
          1585640715000,
          59.86
        ]
      ],
      "expression": "sum:nginx.net.request_per_s{ake_cluster_name:uruai2,host:uruai2-ake-default-b-nlmtind,kube_container_name:nginx,kube_deployment:nginx,kube_namespace:loadtest}",
      "unit": [
        {
          "family": "network",
          "scale_factor": 1,
          "name": "request",
          "short_name": "req",
          "plural": "requests",
          "id": 19
        },
        {
          "family": "time",
          "scale_factor": 1,
          "name": "second",
          "short_name": "s",
          "plural": "seconds",
          "id": 11
        }
      ],
      "display_name": "nginx.net.request_per_s"
    }
  ],
  "to_date": 1585641900000,
  "resp_version": 1,
  "query": "sum:nginx.net.request_per_s{ake_cluster_name:uruai2,kube_container_name:nginx,kube_deployment:nginx,kube_namespace:loadtest}by{host}",
  "message": "",
  "group_by": [
    "host"
  ]
}
			`,
			expected: []datapoint{
				{unixtime: 1585640700, point: 104.03},
				{unixtime: 1585640710, point: 96.5},
				{unixtime: 1585640715, point: 101.65},
			},
		},
	}

	for _, tt := range tests {
		var v interface{}
		if err := json.Unmarshal([]byte(tt.j), &v); err != nil {
			t.Fatal(err)
		}
		dps := mergeAllSeriesDatapointToOne(v)
		fmt.Printf("%#v", dps)
		if !reflect.DeepEqual(dps, tt.expected) {
			t.Fatalf("merged dps is not match (got=%#v, exp=%#v)", dps, tt.expected)
		}
	}
}
