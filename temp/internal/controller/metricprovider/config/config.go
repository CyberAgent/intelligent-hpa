package config

import (
	"fmt"

	ihpav1beta2 "github.com/cyberagent-oss/intelligent-hpa/ihpa-controller/api/v1beta2"
	"github.com/cyberagent-oss/intelligent-hpa/ihpa-controller/controllers/metricprovider"
	datadogmp "github.com/cyberagent-oss/intelligent-hpa/ihpa-controller/controllers/metricprovider/datadog"
	prometheusmp "github.com/cyberagent-oss/intelligent-hpa/ihpa-controller/controllers/metricprovider/prometheus"
)

// MetricProviderConfig holds the configuration for a metric provider.
// It wraps the concrete provider implementations and exposes them
// through a unified interface via ActiveProvider().
type MetricProviderConfig struct {
	Datadog    *datadogmp.Datadog       `json:"datadog,omitempty"`
	Prometheus *prometheusmp.Prometheus `json:"prometheus,omitempty"`
}

// ConvertMetricProvider converts a MetricProvider from the IHPA API
// to a MetricProviderConfig used by FittingJob and Estimator.
func ConvertMetricProvider(mp *ihpav1beta2.MetricProvider) *MetricProviderConfig {
	metricProvider := MetricProviderConfig{}
	if mp.ProviderSource.Datadog != nil {
		datadog := datadogmp.Datadog{
			APIKey:      mp.ProviderSource.Datadog.APIKey,
			APPKey:      mp.ProviderSource.Datadog.APPKey,
			Aggregation: mp.Aggregation,
		}
		metricProvider.Datadog = &datadog
	} else if mp.ProviderSource.Prometheus != nil {
		prometheus := prometheusmp.Prometheus{
			URL:            mp.ProviderSource.Prometheus.URL,
			PushgatewayURL: mp.ProviderSource.Prometheus.PushgatewayURL,
			Aggregation:    mp.Aggregation,
		}
		metricProvider.Prometheus = &prometheus
	}
	return &metricProvider
}

// ActiveProvider returns the active metric provider as a MetricProvider
// interface. Returns nil if no provider is configured.
func (mp *MetricProviderConfig) ActiveProvider() metricprovider.MetricProvider {
	switch {
	case mp.Datadog != nil:
		return mp.Datadog
	case mp.Prometheus != nil:
		return mp.Prometheus
	default:
		return nil
	}
}

// Validate returns an error if no provider is configured.
func (mp *MetricProviderConfig) Validate() error {
	if mp.Datadog == nil && mp.Prometheus == nil {
		return fmt.Errorf("no metric provider is configured")
	}
	return nil
}
