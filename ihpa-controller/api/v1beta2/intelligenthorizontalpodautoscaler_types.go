/*
Copyright 2020 SIA Platform Team.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1beta2

import (
	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// IntelligentHorizontalPodAutoscalerSpec defines the desired state of IntelligentHorizontalPodAutoscaler
type IntelligentHorizontalPodAutoscalerSpec struct {
	// Specifies the horizontalPodAutoscaler(v2beta2) that will be based on ihpa.
	HorizontalPodAutoscalerTemplate ExtendedHorizontalPodAutoscalerTemplateSpec `json:"template"`

	// EstimatorPatchSpec specifies some config for estimator
	EstimatorPatchSpec EstimatorPatchSpec `json:"estimator,omitempty"`

	// MetricProvider is data source and destination of metrics datapoints.
	MetricProvider MetricProvider `json:"metricProvider"`
}

// HorizontalPodAutoscalerTemplateSpec describes the data a HPA should have when created from a template
type ExtendedHorizontalPodAutoscalerTemplateSpec struct {
	// Standard object's metadata of the hpas created from this template.
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Specification of the desired behavior of the hpa.
	Spec ExtendedHorizontalPodAutoscalerSpec `json:"spec,omitempty"`
}

type ExtendedHorizontalPodAutoscalerSpec struct {
	// -- copy from autoscaling.v2beta2 definition --
	ScaleTargetRef autoscalingv2beta2.CrossVersionObjectReference `json:"scaleTargetRef" protobuf:"bytes,1,opt,name=scaleTargetRef"`
	MinReplicas    *int32                                         `json:"minReplicas,omitempty" protobuf:"varint,2,opt,name=minReplicas"`
	MaxReplicas    int32                                          `json:"maxReplicas" protobuf:"varint,3,opt,name=maxReplicas"`
	// but use ExtendedMetricSpec
	Metrics []ExtendedMetricSpec `json:"metrics,omitempty" protobuf:"bytes,4,rep,name=metrics"`
	// ----------------------------------------------
}

// ExtendedMetricSpec is same as autoscaling.v2beta2 but including FittingJob spec
type ExtendedMetricSpec struct {
	// -- copy from autoscaling.v2beta2 definition --
	Type     autoscalingv2beta2.MetricSourceType      `json:"type" protobuf:"bytes,1,name=type"`
	Object   *autoscalingv2beta2.ObjectMetricSource   `json:"object,omitempty" protobuf:"bytes,2,opt,name=object"`
	Pods     *autoscalingv2beta2.PodsMetricSource     `json:"pods,omitempty" protobuf:"bytes,3,opt,name=pods"`
	Resource *autoscalingv2beta2.ResourceMetricSource `json:"resource,omitempty" protobuf:"bytes,4,opt,name=resource"`
	External *autoscalingv2beta2.ExternalMetricSource `json:"external,omitempty" protobuf:"bytes,5,opt,name=external"`
	// ----------------------------------------------

	// FittingJobPatchSpec specifies some config for fittingJob
	FittingJobPatchSpec FittingJobPatchSpec `json:"fittingJob,omitempty"`
}

// MetricSpec extract pure MetricSpec from ExtendedMetricSpec.
func (ems *ExtendedMetricSpec) MetricSpec() *autoscalingv2beta2.MetricSpec {
	return &autoscalingv2beta2.MetricSpec{
		Type:     ems.Type,
		Object:   ems.Object,
		Pods:     ems.Pods,
		Resource: ems.Resource,
		External: ems.External,
	}
}

// FittingJobPatchSpec defines some FittingJobSpec field for patch
type FittingJobPatchSpec struct {
	JobPatchSpec `json:",inline"`

	// Seasonality is time span of bunch metrics period.
	// This is defined as "daily", "weekly", "yearly", "auto".
	// +kubebuilder:validation:Enum=daily;weekly;yearly;auto
	// +kubebuilder:default=auto
	Seasonality string `json:"seasonality,omitempty"`

	// ExecuteOn is hour of executing daily fitting job.
	// Fitting job is executed on at around the hour.
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=23
	// +kubebuilder:default=4
	ExecuteOn int32 `json:"executeOn,omitempty"`

	// ChangePointDetectionConfig is configuration for fittingjob change point detection.
	ChangePointDetectionConfig ChangePointDetectionConfig `json:"changePointDetectionConfig,omitempty"`

	// CustomConfig is custom configurationfor fittingjob.
	CustomConfig string `json:"customConfig,omitempty"`
}

// GenerateFittingJobSpec generate FittingJobSpec from FittingJobPatchSpec
func (fjps *FittingJobPatchSpec) GenerateFittingJobSpec() *FittingJobSpec {
	return &FittingJobSpec{
		JobPatchSpec:               fjps.JobPatchSpec,
		Seasonality:                fjps.Seasonality,
		ExecuteOn:                  fjps.ExecuteOn,
		ChangePointDetectionConfig: fjps.ChangePointDetectionConfig,
		CustomConfig:               fjps.CustomConfig,
	}
}

// EstimatorPatchSpec defines some EstimatorSpec field for patch
type EstimatorPatchSpec struct {
	// Mode is a way to adjust estimate metrics
	// when the metrics out of line.
	// +kubebuilder:validation:Enum=raw;adjust
	// +kubebuilder:default=adjust
	Mode string `json:"mode,omitempty"`

	// GapMinutes is gap time for generating forecast metrics.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=10
	GapMinutes int32 `json:"gapMinutes,omitempty"`
}

// GenerateEstimatorSpec generate EstimatorSpec from EstimatorPatchSpec
func (eps *EstimatorPatchSpec) GenerateEstimatorSpec() *EstimatorSpec {
	return &EstimatorSpec{
		Mode:       eps.Mode,
		GapMinutes: eps.GapMinutes,
	}
}

// MetricProvider defines inforamtion of metrics provider.
type MetricProvider struct {
	// Name is a name of provider
	Name string `json:"name,omitempty"`

	// ProviderSource is source of metrics provider.
	// This is defined as ***ProviderSource.
	ProviderSource `json:",inline"`
}

// ProviderSources defines source of some providers.
type ProviderSource struct {
	Datadog    *DatadogProviderSource    `json:"datadog,omitempty"`
	Prometheus *PrometheusProviderSource `json:"prometheus,omitempty"`
}

// DatadogProviderSource defines parameters for accessing Datadog.
type DatadogProviderSource struct {
	// APIKey is for accessing some function and sending metrics.
	APIKey string `json:"apikey,omitempty"`

	// APPKey is for retrieving metrics.
	APPKey string `json:"appkey,omitempty"`

	// KeysFrom is list from APIKey and APPKey source object.
	// The keys are set by searching "APIKey" and "APPKey" variables.
	KeysFrom []corev1.EnvFromSource `json:"keysFrom,omitempty"`
}

// PrometheusProviderSource defines parameters for accessing Prometheus.
type PrometheusProviderSource struct{}

// IntelligentHorizontalPodAutoscalerStatus defines the observed state of IntelligentHorizontalPodAutoscaler
type IntelligentHorizontalPodAutoscalerStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion

// IntelligentHorizontalPodAutoscaler is the Schema for the intelligenthorizontalpodautoscalers API
// +kubebuilder:resource:shortName=ihpa
type IntelligentHorizontalPodAutoscaler struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IntelligentHorizontalPodAutoscalerSpec   `json:"spec,omitempty"`
	Status IntelligentHorizontalPodAutoscalerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// IntelligentHorizontalPodAutoscalerList contains a list of IntelligentHorizontalPodAutoscaler
type IntelligentHorizontalPodAutoscalerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IntelligentHorizontalPodAutoscaler `json:"items"`
}

func init() {
	SchemeBuilder.Register(&IntelligentHorizontalPodAutoscaler{}, &IntelligentHorizontalPodAutoscalerList{})
}
