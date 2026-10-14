package controllers

import (
	"encoding/json"
	"reflect"
	"testing"

	ihpav1beta2 "github.com/cyberagent-oss/intelligent-hpa/ihpa-controller/api/v1beta2"
	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func testFittingJobSample(t *testing.T) (sample1, sample2 *fittingJobGeneratorImpl) {
	t.Helper()
	sample1 = &fittingJobGeneratorImpl{
		fj: &ihpav1beta2.FittingJob{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "sample1",
				Namespace: "default",
				UID:       "9fc642f3-bb9d-404d-afd5-d04f1d6149ad",
			},
			TypeMeta: metav1.TypeMeta{
				Kind:       "FittingJob",
				APIVersion: "ihpa.ake.cyberagent.co.jp/v1beta2",
			},
			Spec: ihpav1beta2.FittingJobSpec{
				Seasonality: "auto",
				ExecuteOn:   4,
				ChangePointDetectionConfig: ihpav1beta2.ChangePointDetectionConfig{
					PercentageThreshold: 50,
					WindowSize:          100,
					TrajectoryRows:      50,
					TrajectoryFeatures:  5,
					TestRows:            50,
					TestFeatures:        5,
					Lag:                 288,
				},
				CustomConfig: `{"custom_a":1,"custom_b":{"hello":"world"}}`,
				DataConfigMap: corev1.LocalObjectReference{
					Name: "data-configmap",
				},
				JobPatchSpec: ihpav1beta2.JobPatchSpec{
					ImagePullSecrets: []corev1.LocalObjectReference{
						{Name: "pull-secret"},
					},
					Image: "fittingjob-image:v1",
				},
				TargetMetric: autoscalingv2beta2.MetricIdentifier{
					Name: "metric.name",
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"label1": "value1",
							"label2": "value2",
						},
					},
				},
				Provider: ihpav1beta2.MetricProvider{
					Name: "anyprovider",
					ProviderSource: ihpav1beta2.ProviderSource{
						Datadog: &ihpav1beta2.DatadogProviderSource{
							APIKey: "xxx",
							APPKey: "yyy",
						},
					},
				},
			},
		},
	}

	sample2 = &fittingJobGeneratorImpl{
		fj: &ihpav1beta2.FittingJob{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "sample2",
				Namespace: "test",
				UID:       "9fc642f3-bb9d-404d-afd5-d04f1d6149ad",
			},
			TypeMeta: metav1.TypeMeta{
				Kind:       "FittingJob",
				APIVersion: "ihpa.ake.cyberagent.co.jp/v1beta2",
			},
			Spec: ihpav1beta2.FittingJobSpec{
				Seasonality: "daily",
				ExecuteOn:   5,
				DataConfigMap: corev1.LocalObjectReference{
					Name: "data-configmap2",
				},
				JobPatchSpec: ihpav1beta2.JobPatchSpec{
					// related to Job
					ActiveDeadlineSeconds: func(i int64) *int64 { return &i }(300),
					BackoffLimit:          func(i int32) *int32 { return &i }(5),
					Completions:           func(i int32) *int32 { return &i }(10),
					// related to Pod
					ImagePullSecrets: []corev1.LocalObjectReference{{Name: "secret1"}},
					Affinity: &corev1.Affinity{
						PodAntiAffinity: &corev1.PodAntiAffinity{
							PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
								{
									Weight: 50,
									PodAffinityTerm: corev1.PodAffinityTerm{
										TopologyKey: "zone",
										LabelSelector: &metav1.LabelSelector{
											MatchLabels: map[string]string{"app": "a"},
										},
									},
								},
							},
						},
					},
					NodeSelector: map[string]string{"gpu": "true"},
					Tolerations: []corev1.Toleration{
						{
							Key:      "master",
							Operator: "Equal",
							Value:    "true",
							Effect:   "NoSchedule",
						},
					},
					ServiceAccountName: "sa1",
					Volumes: []corev1.Volume{
						{
							Name: "volume1",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{SecretName: "secret1"},
							},
						},
					},
					// related to Container
					Image:           "cyberagentoss/intelligent-hpa-fittingjob:v1",
					ImagePullPolicy: corev1.PullPolicy("Always"),
					Args:            []string{"arg1", "arg2"},
					Command:         []string{"command1"},
					Env: []corev1.EnvVar{
						{Name: "env1", Value: "envval"},
					},
					EnvFrom: []corev1.EnvFromSource{
						{
							ConfigMapRef: &corev1.ConfigMapEnvSource{
								LocalObjectReference: corev1.LocalObjectReference{Name: "cm1"},
							},
						},
					},
					Resources: corev1.ResourceRequirements{
						Limits: corev1.ResourceList{
							corev1.ResourceCPU: *resource.NewScaledQuantity(1000, resource.Milli),
						},
						Requests: corev1.ResourceList{
							corev1.ResourceCPU: *resource.NewScaledQuantity(500, resource.Milli),
						},
					},
				},
				TargetMetric: autoscalingv2beta2.MetricIdentifier{
					Name: "metric.name",
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"l": "v",
						},
					},
				},
				Provider: ihpav1beta2.MetricProvider{
					Name: "datadog",
					ProviderSource: ihpav1beta2.ProviderSource{
						Datadog: &ihpav1beta2.DatadogProviderSource{
							APIKey: "xxx",
							APPKey: "yyy",
						},
					},
				},
			},
		},
	}
	return sample1, sample2
}

func TestFittingJobConfigMapResource(t *testing.T) {
	sample1, sample2 := testFittingJobSample(t)
	tests := []struct {
		base               *fittingJobGeneratorImpl
		expectedObjectMeta metav1.ObjectMeta
		expectedData       map[string]string
	}{
		{
			base: sample1,
			expectedObjectMeta: metav1.ObjectMeta{
				Name:      "sample1-config",
				Namespace: "default",
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion:         "ihpa.ake.cyberagent.co.jp/v1beta2",
						Controller:         func(b bool) *bool { return &b }(true),
						BlockOwnerDeletion: func(b bool) *bool { return &b }(true),
						Kind:               "FittingJob.ihpa.ake.cyberagent.co.jp",
						Name:               "sample1",
						UID:                "9fc642f3-bb9d-404d-afd5-d04f1d6149ad",
					},
				},
			},
			expectedData: map[string]string{
				"config.json": `
				{
					"provider":{
						"datadog":{
							"apikey":"xxx",
							"appkey":"yyy"
						}
					},
					"targetMetricsName":"sum:metric.name",
					"targetTags":{
						"label1":"value1",
						"label2":"value2"
					},
					"seasonality":"auto",
					"dataConfigMapName":"data-configmap",
					"dataConfigMapNamespace":"default",
					"changePointDetection":{
						"percentageThreshold":50,
						"windowSize":100,
						"trajectoryRows":50,
						"trajectoryFeatures":5,
						"testRows":50,
						"testFeatures":5,
						"lag":288
					},
					"customConfig":"{\"custom_a\":1,\"custom_b\":{\"hello\":\"world\"}}"
				}`,
			},
		},
		{
			base: sample2,
			expectedObjectMeta: metav1.ObjectMeta{
				Name:      "sample2-config",
				Namespace: "test",
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion:         "ihpa.ake.cyberagent.co.jp/v1beta2",
						Controller:         func(b bool) *bool { return &b }(true),
						BlockOwnerDeletion: func(b bool) *bool { return &b }(true),
						Kind:               "FittingJob.ihpa.ake.cyberagent.co.jp",
						Name:               "sample2",
						UID:                "9fc642f3-bb9d-404d-afd5-d04f1d6149ad",
					},
				},
			},
			expectedData: map[string]string{
				"config.json": `
				{
					"provider":{
						"datadog":{
							"apikey":"xxx",
							"appkey":"yyy"
						}
					},
					"targetMetricsName":"sum:metric.name",
					"targetTags":{
						"l":"v"
					},
					"seasonality":"daily",
					"dataConfigMapName":"data-configmap2",
					"dataConfigMapNamespace":"test",
					"changePointDetection":{},
					"customConfig":""
				}`,
			},
		},
	}

	for _, tt := range tests {
		got, err := tt.base.ConfigMapResource()
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(got.ObjectMeta, tt.expectedObjectMeta) {
			t.Fatalf("configmap objectmeta is not match (got=%v, exp=%v)", got, tt.expectedObjectMeta)
		}

		configName := "config.json"
		var gotConfig, expConfig interface{}
		if err := json.Unmarshal([]byte(got.Data[configName]), &gotConfig); err != nil {
			t.Fatalf("got data cannot be unmarshal (got=%v): %v", got.Data[configName], err)
		}
		if err := json.Unmarshal([]byte(tt.expectedData[configName]), &expConfig); err != nil {
			t.Fatalf("exp data cannot be unmarshal (exp=%v): %v", tt.expectedData[configName], err)
		}

		if !reflect.DeepEqual(gotConfig, expConfig) {
			t.Fatalf("config is not match (got=%#v, exp=%#v)", gotConfig, expConfig)
		}
	}
}

func TestFittingJobCronJobResource(t *testing.T) {
	sample1, sample2 := testFittingJobSample(t)
	tests := []struct {
		base     *fittingJobGeneratorImpl
		expected *batchv1beta1.CronJob
	}{
		{
			base: sample1,
			expected: &batchv1beta1.CronJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sample1",
					Namespace: "default",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         "ihpa.ake.cyberagent.co.jp/v1beta2",
							Controller:         func(b bool) *bool { return &b }(true),
							BlockOwnerDeletion: func(b bool) *bool { return &b }(true),
							Kind:               "FittingJob.ihpa.ake.cyberagent.co.jp",
							Name:               "sample1",
							UID:                "9fc642f3-bb9d-404d-afd5-d04f1d6149ad",
						},
					},
				},
				Spec: batchv1beta1.CronJobSpec{
					Schedule: "* 4 * * *",
					JobTemplate: batchv1beta1.JobTemplateSpec{
						Spec: batchv1.JobSpec{
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									RestartPolicy: "OnFailure",
									ImagePullSecrets: []corev1.LocalObjectReference{
										{Name: "pull-secret"},
									},
									Containers: []corev1.Container{
										{
											Name:  "fittingjob",
											Image: "fittingjob-image:v1",
											VolumeMounts: []corev1.VolumeMount{
												{
													Name:      "fittingjob-config",
													MountPath: "/fittingjob-config",
												},
											},
										},
									},
									Volumes: []corev1.Volume{
										{
											Name: "fittingjob-config",
											VolumeSource: corev1.VolumeSource{
												ConfigMap: &corev1.ConfigMapVolumeSource{
													LocalObjectReference: corev1.LocalObjectReference{Name: "sample1-config"},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			base: sample2,
			expected: &batchv1beta1.CronJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sample2",
					Namespace: "test",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         "ihpa.ake.cyberagent.co.jp/v1beta2",
							Controller:         func(b bool) *bool { return &b }(true),
							BlockOwnerDeletion: func(b bool) *bool { return &b }(true),
							Kind:               "FittingJob.ihpa.ake.cyberagent.co.jp",
							Name:               "sample2",
							UID:                "9fc642f3-bb9d-404d-afd5-d04f1d6149ad",
						},
					},
				},
				Spec: batchv1beta1.CronJobSpec{
					Schedule: "* 5 * * *",
					JobTemplate: batchv1beta1.JobTemplateSpec{
						Spec: batchv1.JobSpec{
							ActiveDeadlineSeconds: func(i int64) *int64 { return &i }(300),
							BackoffLimit:          func(i int32) *int32 { return &i }(5),
							Completions:           func(i int32) *int32 { return &i }(10),
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									RestartPolicy:      "OnFailure",
									ServiceAccountName: "sa1",
									ImagePullSecrets: []corev1.LocalObjectReference{
										{Name: "secret1"},
									},
									Affinity: &corev1.Affinity{
										PodAntiAffinity: &corev1.PodAntiAffinity{
											PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
												{
													Weight: 50,
													PodAffinityTerm: corev1.PodAffinityTerm{
														TopologyKey: "zone",
														LabelSelector: &metav1.LabelSelector{
															MatchLabels: map[string]string{"app": "a"},
														},
													},
												},
											},
										},
									},
									NodeSelector: map[string]string{"gpu": "true"},
									Tolerations: []corev1.Toleration{
										{
											Key:      "master",
											Operator: "Equal",
											Value:    "true",
											Effect:   "NoSchedule",
										},
									},
									Volumes: []corev1.Volume{
										{
											Name: "volume1",
											VolumeSource: corev1.VolumeSource{
												Secret: &corev1.SecretVolumeSource{SecretName: "secret1"},
											},
										},
										{
											Name: "fittingjob-config",
											VolumeSource: corev1.VolumeSource{
												ConfigMap: &corev1.ConfigMapVolumeSource{
													LocalObjectReference: corev1.LocalObjectReference{Name: "sample2-config"},
												},
											},
										},
									},
									Containers: []corev1.Container{
										{
											Name:            "fittingjob",
											Image:           "cyberagentoss/intelligent-hpa-fittingjob:v1",
											ImagePullPolicy: corev1.PullPolicy("Always"),
											Args:            []string{"arg1", "arg2"},
											Command:         []string{"command1"},
											Env: []corev1.EnvVar{
												{Name: "env1", Value: "envval"},
											},
											EnvFrom: []corev1.EnvFromSource{
												{
													ConfigMapRef: &corev1.ConfigMapEnvSource{
														LocalObjectReference: corev1.LocalObjectReference{Name: "cm1"},
													},
												},
											},
											Resources: corev1.ResourceRequirements{
												Limits: corev1.ResourceList{
													corev1.ResourceCPU: *resource.NewScaledQuantity(1000, resource.Milli),
												},
												Requests: corev1.ResourceList{
													corev1.ResourceCPU: *resource.NewScaledQuantity(500, resource.Milli),
												},
											},
											VolumeMounts: []corev1.VolumeMount{
												{
													Name:      "volume1",
													MountPath: "/volume1",
												},
												{
													Name:      "fittingjob-config",
													MountPath: "/fittingjob-config",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		got, err := tt.base.CronJobResource()
		if err != nil {
			t.Fatal(err)
		}

		got.Spec.Schedule, err = testReplaceCronMinuteSectionToAny(t, got.Spec.Schedule)
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(got, tt.expected) {
			t.Fatalf("cronjob is not match (got=%#v, exp=%#v)", got, tt.expected)
		}
	}
}
