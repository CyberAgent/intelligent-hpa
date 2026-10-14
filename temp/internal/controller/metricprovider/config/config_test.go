package config

import (
	"reflect"
	"testing"

	ihpav1beta2 "github.com/cyberagent-oss/intelligent-hpa/ihpa-controller/api/v1beta2"
	datadogmp "github.com/cyberagent-oss/intelligent-hpa/ihpa-controller/controllers/metricprovider/datadog"
	prometheusmp "github.com/cyberagent-oss/intelligent-hpa/ihpa-controller/controllers/metricprovider/prometheus"
)

func TestConvertMetricProvider(t *testing.T) {
	tests := []struct {
		name     string
		input    *ihpav1beta2.MetricProvider
		expected *MetricProviderConfig
	}{
		{
			name: "datadog without aggregation",
			input: &ihpav1beta2.MetricProvider{
				Name: "datadog",
				ProviderSource: ihpav1beta2.ProviderSource{
					Datadog: &ihpav1beta2.DatadogProviderSource{APIKey: "xxx", APPKey: "yyy"},
				},
			},
			expected: &MetricProviderConfig{
				Datadog: &datadogmp.Datadog{APIKey: "xxx", APPKey: "yyy"},
			},
		},
		{
			name: "datadog with aggregation",
			input: &ihpav1beta2.MetricProvider{
				Name:        "datadog",
				Aggregation: "avg",
				ProviderSource: ihpav1beta2.ProviderSource{
					Datadog: &ihpav1beta2.DatadogProviderSource{APIKey: "xxx", APPKey: "yyy"},
				},
			},
			expected: &MetricProviderConfig{
				Datadog: &datadogmp.Datadog{APIKey: "xxx", APPKey: "yyy", Aggregation: "avg"},
			},
		},
		{
			name: "prometheus without aggregation",
			input: &ihpav1beta2.MetricProvider{
				Name: "prometheus",
				ProviderSource: ihpav1beta2.ProviderSource{
					Prometheus: &ihpav1beta2.PrometheusProviderSource{
						URL:            "http://prometheus:9090",
						PushgatewayURL: "http://pushgateway:9091",
					},
				},
			},
			expected: &MetricProviderConfig{
				Prometheus: &prometheusmp.Prometheus{
					URL:            "http://prometheus:9090",
					PushgatewayURL: "http://pushgateway:9091",
				},
			},
		},
		{
			name: "prometheus with aggregation",
			input: &ihpav1beta2.MetricProvider{
				Name:        "prometheus",
				Aggregation: "max",
				ProviderSource: ihpav1beta2.ProviderSource{
					Prometheus: &ihpav1beta2.PrometheusProviderSource{
						URL:            "http://prometheus:9090",
						PushgatewayURL: "http://pushgateway:9091",
					},
				},
			},
			expected: &MetricProviderConfig{
				Prometheus: &prometheusmp.Prometheus{
					URL:            "http://prometheus:9090",
					PushgatewayURL: "http://pushgateway:9091",
					Aggregation:    "max",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ConvertMetricProvider(tt.input)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Fatalf("metric provider is not match (got=%v, expected=%v)", got, tt.expected)
			}
		})
	}
}

func TestActiveProvider(t *testing.T) {
	tests := []struct {
		name     string
		mp       *MetricProviderConfig
		isNil    bool
	}{
		{
			name: "datadog provider",
			mp: &MetricProviderConfig{
				Datadog: &datadogmp.Datadog{APIKey: "xxx", APPKey: "yyy"},
			},
			isNil: false,
		},
		{
			name: "prometheus provider",
			mp: &MetricProviderConfig{
				Prometheus: &prometheusmp.Prometheus{URL: "http://prometheus:9090"},
			},
			isNil: false,
		},
		{
			name:  "no provider",
			mp:    &MetricProviderConfig{},
			isNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.mp.ActiveProvider()
			if tt.isNil && got != nil {
				t.Fatalf("expected nil, got %v", got)
			}
			if !tt.isNil && got == nil {
				t.Fatalf("expected non-nil, got nil")
			}
		})
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		mp      *MetricProviderConfig
		wantErr bool
	}{
		{
			name: "datadog provider valid",
			mp: &MetricProviderConfig{
				Datadog: &datadogmp.Datadog{APIKey: "xxx", APPKey: "yyy"},
			},
			wantErr: false,
		},
		{
			name: "prometheus provider valid",
			mp: &MetricProviderConfig{
				Prometheus: &prometheusmp.Prometheus{URL: "http://prometheus:9090"},
			},
			wantErr: false,
		},
		{
			name:    "no provider invalid",
			mp:      &MetricProviderConfig{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.mp.Validate()
			if tt.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}
