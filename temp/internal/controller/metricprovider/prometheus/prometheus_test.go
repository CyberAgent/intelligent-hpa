package prometheus

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestPrometheusSend(t *testing.T) {
	var receivedBody string
	var receivedPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		body := make([]byte, 1024)
		n, _ := r.Body.Read(body)
		receivedBody = string(body[:n])
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	p := &Prometheus{
		URL:            "http://prometheus:9090",
		PushgatewayURL: server.URL,
	}

	err := p.Send("ihpa.test.metric", 1609459200, 42.5, []string{"tag1:value1", "tag2:value2"}, nil)
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	if receivedPath != "/metrics/job/estimator" {
		t.Fatalf("unexpected path: got=%s, exp=/metrics/job/estimator", receivedPath)
	}

	expectedBody := `ihpa_test_metric{tag1="value1",tag2="value2"} 42.500000` + "\n"
	if receivedBody != expectedBody {
		t.Fatalf("unexpected body: got=%q, exp=%q", receivedBody, expectedBody)
	}
}

func TestPrometheusSendNoPushgatewayURL(t *testing.T) {
	p := &Prometheus{
		URL: "http://prometheus:9090",
	}

	err := p.Send("ihpa.test.metric", 1609459200, 42.5, []string{"tag1:value1"}, nil)
	if err == nil {
		t.Fatal("expected error when pushgatewayURL is not configured")
	}
}

func TestPrometheusSendError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal server error"))
	}))
	defer server.Close()

	p := &Prometheus{
		PushgatewayURL: server.URL,
	}

	err := p.Send("ihpa.test.metric", 1609459200, 42.5, []string{"tag1:value1"}, nil)
	if err == nil {
		t.Fatal("expected error on server error response")
	}
}

func TestPrometheusFetch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/query" {
			t.Fatalf("unexpected path: got=%s, exp=/api/v1/query", r.URL.Path)
		}

		query := r.URL.Query().Get("query")
		expectedQuery := `container_cpu_usage_seconds_total{host="server1"}`
		if query != expectedQuery {
			t.Fatalf("unexpected query: got=%s, exp=%s", query, expectedQuery)
		}

		timeParam := r.URL.Query().Get("time")
		if timeParam != "1609459200" {
			t.Fatalf("unexpected time: got=%s, exp=1609459200", timeParam)
		}

		resp := queryResponse{
			Status: "success",
			Data: queryData{
				ResultType: "vector",
				Result: []queryResult{
					{
						Metric: map[string]string{"host": "server1"},
						Value:  []interface{}{1609459200, "42.5"},
					},
					{
						Metric: map[string]string{"host": "server2"},
						Value:  []interface{}{1609459200, "10.0"},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := &Prometheus{
		URL: server.URL,
	}

	val, err := p.Fetch("container_cpu_usage_seconds_total", 1609459200, []string{"host:server1"}, nil)
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	expected := 52.5
	if val != expected {
		t.Fatalf("unexpected value: got=%f, exp=%f", val, expected)
	}
}

func TestPrometheusFetchWithAggregator(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("query")
		// Labels must be applied to the inner metric, not outside the aggregator.
		expectedQuery := `sum(container_cpu_usage_seconds_total{namespace="ihpa-test"})`
		if query != expectedQuery {
			t.Fatalf("unexpected query: got=%s, exp=%s", query, expectedQuery)
		}

		resp := queryResponse{
			Status: "success",
			Data: queryData{
				ResultType: "vector",
				Result: []queryResult{
					{
						Metric: map[string]string{"namespace": "ihpa-test"},
						Value:  []interface{}{1609459200, "42.5"},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := &Prometheus{
		URL: server.URL,
	}

	// AddAggregator wraps the metric name as "sum(container_cpu_usage_seconds_total)".
	aggregated := p.AddAggregator("container_cpu_usage_seconds_total")
	val, err := p.Fetch(aggregated, 1609459200, []string{"namespace:ihpa-test"}, nil)
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	if val != 42.5 {
		t.Fatalf("unexpected value: got=%f, exp=42.5", val)
	}
}

func TestBuildPromQL(t *testing.T) {
	tests := []struct {
		name       string
		metricName string
		tags       []string
		expected   string
	}{
		{
			name:       "plain metric with tags",
			metricName: "container_cpu_usage_seconds_total",
			tags:       []string{"namespace:ihpa-test"},
			expected:   `container_cpu_usage_seconds_total{namespace="ihpa-test"}`,
		},
		{
			name:       "aggregator with tags",
			metricName: "sum(container_cpu_usage_seconds_total)",
			tags:       []string{"namespace:ihpa-test"},
			expected:   `sum(container_cpu_usage_seconds_total{namespace="ihpa-test"})`,
		},
		{
			name:       "range vector with tags",
			metricName: "rate(container_cpu_usage_seconds_total[5m])",
			tags:       []string{"namespace:ihpa-test"},
			expected:   `rate(container_cpu_usage_seconds_total{namespace="ihpa-test"}[5m])`,
		},
		{
			name:       "nested aggregator with range vector",
			metricName: "sum(rate(container_cpu_usage_seconds_total[5m]))",
			tags:       []string{"namespace:ihpa-test"},
			expected:   `sum(rate(container_cpu_usage_seconds_total{namespace="ihpa-test"}[5m]))`,
		},
		{
			name:       "no tags returns metric as-is",
			metricName: "container_cpu_usage_seconds_total",
			tags:       nil,
			expected:   "container_cpu_usage_seconds_total",
		},
		{
			name:       "multiple tags",
			metricName: "sum(container_memory_working_set_bytes)",
			tags:       []string{"namespace:ihpa-test", "pod:nginx-abc"},
			expected:   `sum(container_memory_working_set_bytes{namespace="ihpa-test",pod="nginx-abc"})`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildPromQL(tt.metricName, tt.tags)
			if got != tt.expected {
				t.Fatalf("buildPromQL(%q, %v) = %q, expected %q", tt.metricName, tt.tags, got, tt.expected)
			}
		})
	}
}

func TestPrometheusFetchNoURL(t *testing.T) {
	p := &Prometheus{}

	_, err := p.Fetch("metric", 1609459200, nil, nil)
	if err == nil {
		t.Fatal("expected error when URL is not configured")
	}
}

func TestPrometheusFetchEmptyResult(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := queryResponse{
			Status: "success",
			Data: queryData{
				ResultType: "vector",
				Result:     []queryResult{},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := &Prometheus{
		URL: server.URL,
	}

	val, err := p.Fetch("metric", 1609459200, nil, nil)
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	if val != 0.0 {
		t.Fatalf("unexpected value: got=%f, exp=0.0", val)
	}
}

func TestPrometheusFetchError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("bad request"))
	}))
	defer server.Close()

	p := &Prometheus{
		URL: server.URL,
	}

	_, err := p.Fetch("metric", 1609459200, nil, nil)
	if err == nil {
		t.Fatal("expected error on server error response")
	}
}

func TestPrometheusConvertResourceMetricName(t *testing.T) {
	p := &Prometheus{}

	tests := []struct {
		metricName string
		reverse    bool
		expected   *metricIdentifier
	}{
		{
			metricName: "cpu",
			reverse:    false,
			expected:   &metricIdentifier{name: "container_cpu_usage_seconds_total", scale: -9},
		},
		{
			metricName: "memory",
			reverse:    false,
			expected:   &metricIdentifier{name: "container_memory_working_set_bytes", scale: 0},
		},
		{
			metricName: "unknown",
			reverse:    false,
			expected:   nil,
		},
		{
			metricName: "container_cpu_usage_seconds_total",
			reverse:    true,
			expected:   &metricIdentifier{name: "cpu", scale: -9},
		},
		{
			metricName: "container_memory_working_set_bytes",
			reverse:    true,
			expected:   &metricIdentifier{name: "memory", scale: 0},
		},
		{
			metricName: "unknown",
			reverse:    true,
			expected:   nil,
		},
	}

	for _, tt := range tests {
		got := p.ConvertResourceMetricName(tt.metricName, tt.reverse)
		if tt.expected == nil {
			if got != nil {
				t.Fatalf("ConvertResourceMetricName(%q, %v) = %#v, expected nil", tt.metricName, tt.reverse, got)
			}
		} else {
			if got == nil {
				t.Fatalf("ConvertResourceMetricName(%q, %v) = nil, expected %#v", tt.metricName, tt.reverse, tt.expected)
			} else if !reflect.DeepEqual(*got.(*metricIdentifier), *tt.expected) {
				t.Fatalf("ConvertResourceMetricName(%q, %v) = %#v, expected %#v", tt.metricName, tt.reverse, got, tt.expected)
			}
		}
	}
}

func TestPrometheusAddAggregator(t *testing.T) {
	tests := []struct {
		name        string
		aggregation string
		metricName  string
		expected    string
	}{
		{
			name:       "default sum",
			metricName: "container_cpu_usage_seconds_total",
			expected:   "sum(container_cpu_usage_seconds_total)",
		},
		{
			name:       "default sum for arbitrary metric",
			metricName: "ihpa.test.metric",
			expected:   "sum(ihpa.test.metric)",
		},
		{
			name:        "explicit sum",
			aggregation: "sum",
			metricName:  "container_cpu_usage_seconds_total",
			expected:    "sum(container_cpu_usage_seconds_total)",
		},
		{
			name:        "min aggregation",
			aggregation: "min",
			metricName:  "container_cpu_usage_seconds_total",
			expected:    "min(container_cpu_usage_seconds_total)",
		},
		{
			name:        "max aggregation",
			aggregation: "max",
			metricName:  "ihpa.test.metric",
			expected:    "max(ihpa.test.metric)",
		},
		{
			name:        "count aggregation",
			aggregation: "count",
			metricName:  "ihpa.test.metric",
			expected:    "count(ihpa.test.metric)",
		},
		{
			name:        "avg aggregation",
			aggregation: "avg",
			metricName:  "ihpa.test.metric",
			expected:    "avg(ihpa.test.metric)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Prometheus{Aggregation: tt.aggregation}
			got := p.AddAggregator(tt.metricName)
			if got != tt.expected {
				t.Fatalf("AddAggregator(%q) = %q, expected %q", tt.metricName, got, tt.expected)
			}
		})
	}
}
