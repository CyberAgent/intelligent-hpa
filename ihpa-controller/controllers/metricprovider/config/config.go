package config

import (
	ihpav1beta2 "github.com/cyberagent-oss/intelligent-hpa/ihpa-controller/api/v1beta2"
	"github.com/cyberagent-oss/intelligent-hpa/ihpa-controller/controllers/metricprovider"
	datadogmp "github.com/cyberagent-oss/intelligent-hpa/ihpa-controller/controllers/metricprovider/datadog"
	prometheusmp "github.com/cyberagent-oss/intelligent-hpa/ihpa-controller/controllers/metricprovider/prometheus"
)

type MetricProviderConfig struct {
	Datadog    *datadogmp.Datadog       `json:"datadog,omitempty"`
	Prometheus *prometheusmp.Prometheus `json:"prometheus,omitempty"`
}

// convertMetricProvider converts MetricProvider which is defined for
// IHPA api to MerticProvider which is defined for FittingJob.
func ConvertMetricProvider(mp *ihpav1beta2.MetricProvider) *MetricProviderConfig {
	metricProvider := MetricProviderConfig{}
	if mp.ProviderSource.Datadog != nil {
		datadog := datadogmp.Datadog{
			APIKey: mp.ProviderSource.Datadog.APIKey,
			APPKey: mp.ProviderSource.Datadog.APPKey,
		}
		metricProvider.Datadog = &datadog
	} else if mp.ProviderSource.Prometheus != nil {
		prometheus := prometheusmp.Prometheus{}
		metricProvider.Prometheus = &prometheus
	}
	return &metricProvider
}

// TODO: bad code because of bad struct metricProvider
func (mp *MetricProviderConfig) ActiveProvider() metricprovider.MetricProvider {
	if mp.Datadog != nil {
		return mp.Datadog
	} else if mp.Prometheus != nil {
		return mp.Prometheus
	}
	return nil
}
