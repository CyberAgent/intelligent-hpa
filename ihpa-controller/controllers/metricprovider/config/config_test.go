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
		input    *ihpav1beta2.MetricProvider
		expected *MetricProviderConfig
	}{
		{
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
			input: &ihpav1beta2.MetricProvider{
				Name: "prometheus",
				ProviderSource: ihpav1beta2.ProviderSource{
					Prometheus: &ihpav1beta2.PrometheusProviderSource{},
				},
			},
			expected: &MetricProviderConfig{
				Prometheus: &prometheusmp.Prometheus{},
			},
		},
	}

	for _, tt := range tests {
		got := ConvertMetricProvider(tt.input)
		if !reflect.DeepEqual(got, tt.expected) {
			t.Fatalf("metric provider is not match (got=%v, expected=%v)", got, tt.expected)
		}
	}
}
