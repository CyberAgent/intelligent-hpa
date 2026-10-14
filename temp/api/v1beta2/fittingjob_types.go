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
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// FittingJobSpec defines the desired state of FittingJob
type FittingJobSpec struct {
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

	// DataConfigMap is destination of result fittingjob forecasted.
	DataConfigMap corev1.LocalObjectReference `json:"dataConfigMap,omitempty"`

	// TargetMetric is a metric identifier for forecast target.
	TargetMetric autoscalingv2beta2.MetricIdentifier `json:"metric,omitempty"`

	// Provider is a metricProvider for fetching target metric.
	Provider MetricProvider `json:"provider,omitempty"`
}

// JobPatchSpec defines some JobSpec field for patch
type JobPatchSpec struct {
	// related to Job
	ActiveDeadlineSeconds *int64 `json:"activeDeadlineSeconds,omitempty"`
	BackoffLimit          *int32 `json:"backoffLimit,omitempty"`
	Completions           *int32 `json:"completions,omitempty"`

	// related to Pod
	Affinity           *corev1.Affinity              `json:"affinity,omitempty"`
	ImagePullSecrets   []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`
	NodeSelector       map[string]string             `json:"nodeSelector,omitempty"`
	ServiceAccountName string                        `json:"serviceAccountName,omitempty"`
	Tolerations        []corev1.Toleration           `json:"tolerations,omitempty"`
	Volumes            []corev1.Volume               `json:"volumes,omitempty"`
	// volume name is used as mount path

	// related to Container
	Args            []string                    `json:"args,omitempty"`
	Command         []string                    `json:"command,omitempty"`
	Env             []corev1.EnvVar             `json:"env,omitempty"`
	EnvFrom         []corev1.EnvFromSource      `json:"envFrom,omitempty"`
	Image           string                      `json:"image,omitempty"`
	ImagePullPolicy corev1.PullPolicy           `json:"imagePullPolicy,omitempty"`
	Resources       corev1.ResourceRequirements `json:"resources,omitempty"`
}

// GenerateJobSpec generate JobSpec from JobPatchSpec
func (jps *JobPatchSpec) GenerateJobSpec() *batchv1.JobSpec {
	volumeMounts := make([]corev1.VolumeMount, 0, len(jps.Volumes))
	for _, volume := range jps.Volumes {
		volumeMount := corev1.VolumeMount{
			Name:      volume.Name,
			MountPath: "/" + volume.Name,
		}
		volumeMounts = append(volumeMounts, volumeMount)
	}

	return &batchv1.JobSpec{
		ActiveDeadlineSeconds: jps.ActiveDeadlineSeconds,
		BackoffLimit:          jps.BackoffLimit,
		Completions:           jps.Completions,
		Template: corev1.PodTemplateSpec{
			Spec: corev1.PodSpec{
				Affinity:           jps.Affinity,
				ImagePullSecrets:   jps.ImagePullSecrets,
				NodeSelector:       jps.NodeSelector,
				ServiceAccountName: jps.ServiceAccountName,
				Tolerations:        jps.Tolerations,
				Volumes:            jps.Volumes,
				Containers: []corev1.Container{
					{
						Args:            jps.Args,
						Command:         jps.Command,
						Env:             jps.Env,
						EnvFrom:         jps.EnvFrom,
						Image:           jps.Image,
						ImagePullPolicy: jps.ImagePullPolicy,
						Resources:       jps.Resources,
						VolumeMounts:    volumeMounts,
					},
				},
			},
		},
	}
}

// ChangePointDetectionConfig is configuration for fittingjob change point detection.
type ChangePointDetectionConfig struct {
	// PercentageThreshold is threshold of anormaly detection for
	// selection of training data in percentage.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=99
	// +kubebuilder:default=50
	PercentageThreshold int32 `json:"percentageThreshold,omitempty"`

	// WindowSize is width of sliding window for partial time series.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=100
	WindowSize int32 `json:"windowSize,omitempty"`

	// TrajectoryRows is rows of trajectory matrix.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=50
	TrajectoryRows int32 `json:"trajectoryRows,omitempty"`

	// TrajectoryFeatures is number of trajectory features of left singular vectors.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=5
	TrajectoryFeatures int32 `json:"trajectoryFeatures,omitempty"`

	// TestRows is rows of test matrix
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=50
	TestRows int32 `json:"testRows,omitempty"`

	// TestFeatures is number of test features of left singular vectors.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=5
	TestFeatures int32 `json:"testFeatures,omitempty"`

	// Lag is gap of trajectory and test.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=288
	Lag int32 `json:"lag,omitempty"`
}

// FittingJobStatus defines the observed state of FittingJob
type FittingJobStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion

// FittingJob is the Schema for the fittingjobs API
// +kubebuilder:resource:shortName=fj
type FittingJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FittingJobSpec   `json:"spec,omitempty"`
	Status FittingJobStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// FittingJobList contains a list of FittingJob
type FittingJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FittingJob `json:"items"`
}

func init() {
	SchemeBuilder.Register(&FittingJob{}, &FittingJobList{})
}
