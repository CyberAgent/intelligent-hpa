package controllers

import (
	ihpav1beta2 "github.com/cyberagent-oss/intelligent-hpa/ihpa-controller/api/v1beta2"
	mpconfig "github.com/cyberagent-oss/intelligent-hpa/ihpa-controller/controllers/metricprovider/config"
)

// FittingJobConfig is representation of config for fittingJob.
type FittingJobConfig struct {
	MetricProvider    mpconfig.MetricProviderConfig `json:"provider"`
	DumpPath          string                        `json:"dumpPath,omitempty"` // maybe not use
	TargetMetricsName string                        `json:"targetMetricsName"`
	// TargetTags is used for identifying target metrics (not forecasted metrics).
	// IHPA refers to metric provider for retrieving target metrics. If possible to
	// retrieve target metric from metrics-server in k8s, we should get that from there,
	// but, in now, IHPA retrieves any metrics from metric provider for making easy to
	// implement retrieve section (correspond all resource type, Resource, Object, Pods, External)
	TargetTags                 map[string]string                      `json:"targetTags"`
	Seasonality                string                                 `json:"seasonality"`
	DataConfigMapName          string                                 `json:"dataConfigMapName"`
	DataConfigMapNamespace     string                                 `json:"dataConfigMapNamespace"`
	ChangePointDetectionConfig ihpav1beta2.ChangePointDetectionConfig `json:"changePointDetection"`
	CustomConfig               string                                 `json:"customConfig"`
}
