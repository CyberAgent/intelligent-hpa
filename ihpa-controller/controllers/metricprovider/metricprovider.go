package metricprovider

type MetricProvider interface {
	// Send send a metric with tags
	Send(metricName string, timestamp int64, point float64, tags []string, opts map[string]interface{}) error
	// Fetch fetch one metric at timestamp
	Fetch(metricName string, timestamp int64, tags []string, opts map[string]interface{}) (float64, error)
	// ConvertResourceMetricName convert given name to provider depended name for Resource type.
	// If reverse is true, then reverse lookup metricName.
	ConvertResourceMetricName(metricName string, reverse bool) MetricIdentifier
	// ConvertObjectMetricName convert given name to provider depended name for Object type.
	ConvertObjectMetricName(metricName string, reverse bool) MetricIdentifier
	// ConvertPodsMetricName convert given name to provider depended name for Pods type.
	ConvertPodsMetricName(metricName string, reverse bool) MetricIdentifier
	// AddSumAggregator add sum aggregator to metric name to get sum of metric point.
	AddSumAggregator(metricName string) string
}

type MetricIdentifier interface {
	GetName() string
	GetScale() int
}
