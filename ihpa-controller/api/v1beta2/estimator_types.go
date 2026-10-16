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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// EstimatorSpec defines the desired state of Estimator
type EstimatorSpec struct {
	// Mode is a way to adjust estimate metrics
	// when the metrics out of line.
	// +kubebuilder:validation:Enum=raw;adjust
	// +kubebuilder:default=adjust
	Mode string `json:"mode,omitempty"`

	// GapMinutes is gap time for generating forecast metrics.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=10
	GapMinutes int32 `json:"gapMinutes,omitempty"`

	// MetricName is a metric name to send
	MetricName string `json:"metricName"`

	// MetricTags is some tags to apply metric.
	MetricTags []string `json:"metricTags"`

	// BaseMetricName is a metric name to get base metric for adjustment.
	BaseMetricName string `json:"baseMetricName,omitempty"`

	// BaseMetricTags is some tags to get base metric for adjustment.
	BaseMetricTags []string `json:"baseMetricTags,omitempty"`

	// MetricProvider is data source and destination of metrics datapoints.
	Provider MetricProvider `json:"provider"`

	// DataConfigMap is destination of result fittingjob forecasted.
	DataConfigMap corev1.LocalObjectReference `json:"dataConfigMap"`
}

// EstimatorStatus defines the observed state of Estimator
type EstimatorStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true

// Estimator is the Schema for the estimators API
type Estimator struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EstimatorSpec   `json:"spec,omitempty"`
	Status EstimatorStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// EstimatorList contains a list of Estimator
type EstimatorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Estimator `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Estimator{}, &EstimatorList{})
}
