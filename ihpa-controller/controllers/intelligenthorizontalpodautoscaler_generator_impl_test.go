package controllers

import (
	"fmt"
	"reflect"
	"testing"

	ihpav1beta2 "github.com/cyberagent-oss/intelligent-hpa/ihpa-controller/api/v1beta2"
	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func testIHPAGeneratorSample(t *testing.T) (*ihpaGeneratorImpl, *ihpaGeneratorImpl) {
	t.Helper()
	sample1 := &ihpaGeneratorImpl{
		kubeSystemUID: "46f9e396-d3c4-4103-a807-49054f47bbfb",
		scaleTargetRequests: corev1.ResourceList{
			corev1.ResourceCPU: *resource.NewScaledQuantity(500, resource.Milli),
		},
		ihpa: &ihpav1beta2.IntelligentHorizontalPodAutoscaler{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "sample1",
				Namespace: "default",
				UID:       "9fc642f3-bb9d-404d-afd5-d04f1d6149ad",
			},
			TypeMeta: metav1.TypeMeta{
				Kind:       "IntelligentHorizontalPodAutoscaler",
				APIVersion: "ihpa.ake.cyberagent.co.jp/v1beta1",
			},
			Spec: ihpav1beta2.IntelligentHorizontalPodAutoscalerSpec{
				HorizontalPodAutoscalerTemplate: ihpav1beta2.ExtendedHorizontalPodAutoscalerTemplateSpec{
					Spec: ihpav1beta2.ExtendedHorizontalPodAutoscalerSpec{
						ScaleTargetRef: autoscalingv2beta2.CrossVersionObjectReference{
							APIVersion: "apps/v1",
							Kind:       "Deployment",
							Name:       "nginx",
						},
						MinReplicas: func(i int32) *int32 { return &i }(3),
						Metrics: []ihpav1beta2.ExtendedMetricSpec{
							{
								Type: "Resource",
								Resource: &autoscalingv2beta2.ResourceMetricSource{
									Name: "cpu",
									Target: autoscalingv2beta2.MetricTarget{
										Type:               "Utilization",
										AverageUtilization: func(i int32) *int32 { return &i }(50),
									},
								},
								FittingJobPatchSpec: ihpav1beta2.FittingJobPatchSpec{
									Seasonality: "weekly",
									ExecuteOn:   4,
									// omit JobPatch
								},
							},
						},
					},
				},
				EstimatorPatchSpec: ihpav1beta2.EstimatorPatchSpec{
					Mode:       "adjust",
					GapMinutes: 5,
				},
				MetricProvider: ihpav1beta2.MetricProvider{
					Name: "sample-provider",
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
	sample2 := &ihpaGeneratorImpl{
		kubeSystemUID: "9833e04f-a689-47f9-b588-36a616432abd",
		scaleTargetRequests: corev1.ResourceList{
			corev1.ResourceCPU:     *resource.NewScaledQuantity(2, 0),
			corev1.ResourceMemory:  *resource.NewScaledQuantity(500, resource.Mega),
			corev1.ResourceStorage: *resource.NewScaledQuantity(4, resource.Giga),
		},
		ihpa: &ihpav1beta2.IntelligentHorizontalPodAutoscaler{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "sample2",
				Namespace: "web",
				UID:       "9fc642f3-bb9d-404d-afd5-d04f1d6149ad",
			},
			TypeMeta: metav1.TypeMeta{
				Kind:       "IntelligentHorizontalPodAutoscaler",
				APIVersion: "ihpa.ake.cyberagent.co.jp/v1",
			},
			Spec: ihpav1beta2.IntelligentHorizontalPodAutoscalerSpec{
				HorizontalPodAutoscalerTemplate: ihpav1beta2.ExtendedHorizontalPodAutoscalerTemplateSpec{
					Spec: ihpav1beta2.ExtendedHorizontalPodAutoscalerSpec{
						ScaleTargetRef: autoscalingv2beta2.CrossVersionObjectReference{
							APIVersion: "apps/v1",
							Kind:       "StatefulSet",
							Name:       "web-db",
						},
						MinReplicas: func(i int32) *int32 { return &i }(5),
						MaxReplicas: 10,
						Metrics: []ihpav1beta2.ExtendedMetricSpec{
							{
								Type: "Resource",
								Resource: &autoscalingv2beta2.ResourceMetricSource{
									Name: "cpu",
									Target: autoscalingv2beta2.MetricTarget{
										Type:               "Utilization",
										AverageUtilization: func(i int32) *int32 { return &i }(50),
									},
								},
								FittingJobPatchSpec: ihpav1beta2.FittingJobPatchSpec{
									Seasonality: "daily",
									ExecuteOn:   24,
									ChangePointDetectionConfig: ihpav1beta2.ChangePointDetectionConfig{
										PercentageThreshold: 50,
										WindowSize:          100,
										TrajectoryRows:      50,
										TrajectoryFeatures:  5,
										TestRows:            50,
										TestFeatures:        5,
										Lag:                 288,
									},
									CustomConfig: `this is custom config`,
									JobPatchSpec: ihpav1beta2.JobPatchSpec{
										ImagePullSecrets: []corev1.LocalObjectReference{
											{Name: "secret1"},
											{Name: "secret2"},
										},
										Image: "my_image:v1",
									},
								},
							},
							{
								Type: "Resource",
								Resource: &autoscalingv2beta2.ResourceMetricSource{
									Name: "memory",
									Target: autoscalingv2beta2.MetricTarget{
										Type:               "Utilization",
										AverageUtilization: func(i int32) *int32 { return &i }(80),
									},
								},
								// omit FittingJobPatch
							},
							{
								Type: "External",
								External: &autoscalingv2beta2.ExternalMetricSource{
									Metric: autoscalingv2beta2.MetricIdentifier{
										Name: "nginx.net.request_per_s",
										Selector: &metav1.LabelSelector{
											MatchLabels: map[string]string{
												"kube_statefulset": "web-db",
												"kube_namespace":   "web",
												"kube_app":         "sample",
											},
										},
									},
									Target: autoscalingv2beta2.MetricTarget{
										Type:         "AverageValue",
										AverageValue: resource.NewQuantity(50, resource.DecimalSI),
									},
								},
								FittingJobPatchSpec: ihpav1beta2.FittingJobPatchSpec{
									Seasonality: "auto",
									ExecuteOn:   2,
									// full of JobPatch
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
								},
							},
						},
					},
				},
				EstimatorPatchSpec: ihpav1beta2.EstimatorPatchSpec{
					Mode:       "raw",
					GapMinutes: 10,
				},
				MetricProvider: ihpav1beta2.MetricProvider{
					Name: "sample-provider",
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

func TestHorizontalPodAutoscalerResource(t *testing.T) {
	sample1, sample2 := testIHPAGeneratorSample(t)
	tests := []struct {
		base     *ihpaGeneratorImpl
		expected *autoscalingv2beta2.HorizontalPodAutoscaler
	}{
		{
			base: sample1,
			expected: &autoscalingv2beta2.HorizontalPodAutoscaler{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ihpa-sample1",
					Namespace: "default",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         "ihpa.ake.cyberagent.co.jp/v1beta1",
							Controller:         func(b bool) *bool { return &b }(true),
							BlockOwnerDeletion: func(b bool) *bool { return &b }(true),
							Kind:               "IntelligentHorizontalPodAutoscaler.ihpa.ake.cyberagent.co.jp",
							Name:               "sample1",
							UID:                "9fc642f3-bb9d-404d-afd5-d04f1d6149ad",
						},
					},
				},
				Spec: autoscalingv2beta2.HorizontalPodAutoscalerSpec{
					ScaleTargetRef: autoscalingv2beta2.CrossVersionObjectReference{
						APIVersion: "apps/v1",
						Kind:       "Deployment",
						Name:       "nginx",
					},
					MinReplicas: func(i int32) *int32 { return &i }(3),
					Metrics: []autoscalingv2beta2.MetricSpec{
						{
							Type: "Resource",
							Resource: &autoscalingv2beta2.ResourceMetricSource{
								Name: "cpu",
								Target: autoscalingv2beta2.MetricTarget{
									Type:               "Utilization",
									AverageUtilization: func(i int32) *int32 { return &i }(50),
								},
							},
						},
						{
							Type: "External",
							External: &autoscalingv2beta2.ExternalMetricSource{
								Metric: autoscalingv2beta2.MetricIdentifier{
									Name: "ake.ihpa.forecasted_kubernetes_cpu_usage_total",
									Selector: &metav1.LabelSelector{
										MatchLabels: map[string]string{
											"kube_system_uid": "46f9e396-d3c4-4103-a807-49054f47bbfb",
											"kube_deployment": "nginx",
											"kube_namespace":  "default",
										},
									},
								},
								Target: autoscalingv2beta2.MetricTarget{
									Type:         "AverageValue",
									AverageValue: resource.NewQuantity(resource.NewQuantity(250, resource.DecimalSI).ScaledValue(resource.Micro), resource.DecimalSI), // 250M
								},
							},
						},
					},
				},
			},
		},
		{
			base: sample2,
			expected: &autoscalingv2beta2.HorizontalPodAutoscaler{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ihpa-sample2",
					Namespace: "web",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         "ihpa.ake.cyberagent.co.jp/v1",
							Controller:         func(b bool) *bool { return &b }(true),
							BlockOwnerDeletion: func(b bool) *bool { return &b }(true),
							Kind:               "IntelligentHorizontalPodAutoscaler.ihpa.ake.cyberagent.co.jp",
							Name:               "sample2",
							UID:                "9fc642f3-bb9d-404d-afd5-d04f1d6149ad",
						},
					},
				},
				Spec: autoscalingv2beta2.HorizontalPodAutoscalerSpec{
					ScaleTargetRef: autoscalingv2beta2.CrossVersionObjectReference{
						APIVersion: "apps/v1",
						Kind:       "StatefulSet",
						Name:       "web-db",
					},
					MinReplicas: func(i int32) *int32 { return &i }(5),
					MaxReplicas: 10,
					Metrics: []autoscalingv2beta2.MetricSpec{
						{
							Type: "Resource",
							Resource: &autoscalingv2beta2.ResourceMetricSource{
								Name: "cpu",
								Target: autoscalingv2beta2.MetricTarget{
									Type:               "Utilization",
									AverageUtilization: func(i int32) *int32 { return &i }(50),
								},
							},
						},
						{
							Type: "Resource",
							Resource: &autoscalingv2beta2.ResourceMetricSource{
								Name: "memory",
								Target: autoscalingv2beta2.MetricTarget{
									Type:               "Utilization",
									AverageUtilization: func(i int32) *int32 { return &i }(80),
								},
							},
						},
						{
							Type: "External",
							External: &autoscalingv2beta2.ExternalMetricSource{
								Metric: autoscalingv2beta2.MetricIdentifier{
									Name: "nginx.net.request_per_s",
									Selector: &metav1.LabelSelector{
										MatchLabels: map[string]string{
											"kube_statefulset": "web-db",
											"kube_namespace":   "web",
											"kube_app":         "sample",
										},
									},
								},
								Target: autoscalingv2beta2.MetricTarget{
									Type:         "AverageValue",
									AverageValue: resource.NewQuantity(50, resource.DecimalSI),
								},
							},
						},
						{
							Type: "External",
							External: &autoscalingv2beta2.ExternalMetricSource{
								Metric: autoscalingv2beta2.MetricIdentifier{
									Name: "ake.ihpa.forecasted_kubernetes_cpu_usage_total",
									Selector: &metav1.LabelSelector{
										MatchLabels: map[string]string{
											"kube_system_uid":  "9833e04f-a689-47f9-b588-36a616432abd",
											"kube_statefulset": "web-db",
											"kube_namespace":   "web",
										},
									},
								},
								Target: autoscalingv2beta2.MetricTarget{
									Type:         "AverageValue",
									AverageValue: resource.NewQuantity(resource.NewQuantity(1000, resource.DecimalSI).ScaledValue(resource.Micro), resource.DecimalSI),
									// 1000M (1,000,000,000) (resource.Micro = 10^6)
								},
							},
						},
						{
							Type: "External",
							External: &autoscalingv2beta2.ExternalMetricSource{
								Metric: autoscalingv2beta2.MetricIdentifier{
									Name: "ake.ihpa.forecasted_kubernetes_memory_usage",
									Selector: &metav1.LabelSelector{
										MatchLabels: map[string]string{
											"kube_system_uid":  "9833e04f-a689-47f9-b588-36a616432abd",
											"kube_statefulset": "web-db",
											"kube_namespace":   "web",
										},
									},
								},
								Target: autoscalingv2beta2.MetricTarget{
									Type:         "AverageValue",
									AverageValue: resource.NewQuantity(resource.NewQuantity(400_000_000, resource.DecimalSI).Value(), resource.DecimalSI),
								},
							},
						},
						{
							Type: "External",
							External: &autoscalingv2beta2.ExternalMetricSource{
								Metric: autoscalingv2beta2.MetricIdentifier{
									Name: "ake.ihpa.forecasted_nginx_net_request_per_s",
									Selector: &metav1.LabelSelector{
										MatchLabels: map[string]string{
											"kube_system_uid":  "9833e04f-a689-47f9-b588-36a616432abd",
											"kube_statefulset": "web-db",
											"kube_namespace":   "web",
										},
									},
								},
								Target: autoscalingv2beta2.MetricTarget{
									Type:         "AverageValue",
									AverageValue: resource.NewQuantity(50, resource.DecimalSI),
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		got, err := tt.base.HorizontalPodAutoscalerResource()
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(got, tt.expected) {
			for i, target := range []*autoscalingv2beta2.HorizontalPodAutoscaler{got, tt.expected} {
				for _, m := range target.Spec.Metrics {
					if ext := m.External; ext != nil {
						var prefix string
						if i == 0 {
							prefix = "[got]"
						} else {
							prefix = "[exp]"
						}

						fmt.Printf("%s %s: val(%#v), avgval(%#v)\n", prefix, ext.Metric.Name, ext.Target.Value, ext.Target.AverageValue)
					}
				}
			}

			t.Fatalf("hpa resource is not match\ngot=%#v\nexp=%#v", got, tt.expected)
		}
	}
}

func TestFittingJobResources(t *testing.T) {
	sample1, sample2 := testIHPAGeneratorSample(t)
	tests := []struct {
		base     *ihpaGeneratorImpl
		expected []ihpav1beta2.FittingJob
	}{
		{
			base: sample1,
			expected: []ihpav1beta2.FittingJob{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ihpa-sample1-cpu",
						Namespace: "default",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion:         "ihpa.ake.cyberagent.co.jp/v1beta1",
								Controller:         func(b bool) *bool { return &b }(true),
								BlockOwnerDeletion: func(b bool) *bool { return &b }(true),
								Kind:               "IntelligentHorizontalPodAutoscaler.ihpa.ake.cyberagent.co.jp",
								Name:               "sample1",
								UID:                "9fc642f3-bb9d-404d-afd5-d04f1d6149ad",
							},
						},
						Annotations: map[string]string{
							"ihpa.ake.cyberagent.co.jp/fittingjob-id": "b9aa658a6b30a452ee08d0fbea2c4f40",
						},
					},
					Spec: ihpav1beta2.FittingJobSpec{
						Seasonality:   "weekly",
						ExecuteOn:     4,
						DataConfigMap: corev1.LocalObjectReference{Name: "ihpa-sample1-cpu"},
						JobPatchSpec: ihpav1beta2.JobPatchSpec{
							ServiceAccountName: "ihpa-sample1",
						},
						TargetMetric: autoscalingv2beta2.MetricIdentifier{
							Name: "kubernetes.cpu.usage.total",
							Selector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"kube_system_uid": "46f9e396-d3c4-4103-a807-49054f47bbfb",
									"kube_deployment": "nginx",
									"kube_namespace":  "default",
								},
							},
						},
						Provider: ihpav1beta2.MetricProvider{
							Name: "sample-provider",
							ProviderSource: ihpav1beta2.ProviderSource{
								Datadog: &ihpav1beta2.DatadogProviderSource{
									APIKey: "xxx",
									APPKey: "yyy",
								},
							},
						},
					},
				},
			},
		},
		{
			base: sample2,
			expected: []ihpav1beta2.FittingJob{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ihpa-sample2-cpu",
						Namespace: "web",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion:         "ihpa.ake.cyberagent.co.jp/v1",
								Controller:         func(b bool) *bool { return &b }(true),
								BlockOwnerDeletion: func(b bool) *bool { return &b }(true),
								Kind:               "IntelligentHorizontalPodAutoscaler.ihpa.ake.cyberagent.co.jp",
								Name:               "sample2",
								UID:                "9fc642f3-bb9d-404d-afd5-d04f1d6149ad",
							},
						},
						Annotations: map[string]string{
							"ihpa.ake.cyberagent.co.jp/fittingjob-id": "69f9447bad75992b533334827ce092ad",
						},
					},
					Spec: ihpav1beta2.FittingJobSpec{
						Seasonality: "daily",
						ExecuteOn:   24,
						ChangePointDetectionConfig: ihpav1beta2.ChangePointDetectionConfig{
							PercentageThreshold: 50,
							WindowSize:          100,
							TrajectoryRows:      50,
							TrajectoryFeatures:  5,
							TestRows:            50,
							TestFeatures:        5,
							Lag:                 288,
						},
						CustomConfig:  `this is custom config`,
						DataConfigMap: corev1.LocalObjectReference{Name: "ihpa-sample2-cpu"},
						JobPatchSpec: ihpav1beta2.JobPatchSpec{
							Image: "my_image:v1",
							ImagePullSecrets: []corev1.LocalObjectReference{
								{Name: "secret1"},
								{Name: "secret2"},
							},
							ServiceAccountName: "ihpa-sample2",
						},
						TargetMetric: autoscalingv2beta2.MetricIdentifier{
							Name: "kubernetes.cpu.usage.total",
							Selector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"kube_system_uid":  "9833e04f-a689-47f9-b588-36a616432abd",
									"kube_statefulset": "web-db",
									"kube_namespace":   "web",
								},
							},
						},
						Provider: ihpav1beta2.MetricProvider{
							Name: "sample-provider",
							ProviderSource: ihpav1beta2.ProviderSource{
								Datadog: &ihpav1beta2.DatadogProviderSource{
									APIKey: "xxx",
									APPKey: "yyy",
								},
							},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ihpa-sample2-memory",
						Namespace: "web",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion:         "ihpa.ake.cyberagent.co.jp/v1",
								Controller:         func(b bool) *bool { return &b }(true),
								BlockOwnerDeletion: func(b bool) *bool { return &b }(true),
								Kind:               "IntelligentHorizontalPodAutoscaler.ihpa.ake.cyberagent.co.jp",
								Name:               "sample2",
								UID:                "9fc642f3-bb9d-404d-afd5-d04f1d6149ad",
							},
						},
						Annotations: map[string]string{
							"ihpa.ake.cyberagent.co.jp/fittingjob-id": "c5abea3be857f41dc43ed7dec2eee7e3",
						},
					},
					Spec: ihpav1beta2.FittingJobSpec{
						DataConfigMap: corev1.LocalObjectReference{Name: "ihpa-sample2-memory"},
						JobPatchSpec: ihpav1beta2.JobPatchSpec{
							ServiceAccountName: "ihpa-sample2",
						},
						TargetMetric: autoscalingv2beta2.MetricIdentifier{
							Name: "kubernetes.memory.usage",
							Selector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"kube_system_uid":  "9833e04f-a689-47f9-b588-36a616432abd",
									"kube_statefulset": "web-db",
									"kube_namespace":   "web",
								},
							},
						},
						Provider: ihpav1beta2.MetricProvider{
							Name: "sample-provider",
							ProviderSource: ihpav1beta2.ProviderSource{
								Datadog: &ihpav1beta2.DatadogProviderSource{
									APIKey: "xxx",
									APPKey: "yyy",
								},
							},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ihpa-sample2-nginx-net-request-per-s",
						Namespace: "web",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion:         "ihpa.ake.cyberagent.co.jp/v1",
								Controller:         func(b bool) *bool { return &b }(true),
								BlockOwnerDeletion: func(b bool) *bool { return &b }(true),
								Kind:               "IntelligentHorizontalPodAutoscaler.ihpa.ake.cyberagent.co.jp",
								Name:               "sample2",
								UID:                "9fc642f3-bb9d-404d-afd5-d04f1d6149ad",
							},
						},
						Annotations: map[string]string{
							"ihpa.ake.cyberagent.co.jp/fittingjob-id": "e9932760a233dffd256e6e0de941096e",
						},
					},
					Spec: ihpav1beta2.FittingJobSpec{
						Seasonality:   "auto",
						ExecuteOn:     2,
						DataConfigMap: corev1.LocalObjectReference{Name: "ihpa-sample2-nginx-net-request-per-s"},
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
							Name: "nginx.net.request_per_s",
							Selector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"kube_statefulset": "web-db",
									"kube_namespace":   "web",
									"kube_app":         "sample",
								},
							},
						},
						Provider: ihpav1beta2.MetricProvider{
							Name: "sample-provider",
							ProviderSource: ihpav1beta2.ProviderSource{
								Datadog: &ihpav1beta2.DatadogProviderSource{
									APIKey: "xxx",
									APPKey: "yyy",
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		got, err := tt.base.FittingJobResources()
		if err != nil {
			t.Fatal(err)
		}
		for i := range got {
			if !reflect.DeepEqual(*(got[i]), tt.expected[i]) {
				t.Fatalf("fittingjob resources are not match\ngot=%#v\nexp=%#v", *(got[i]), tt.expected[i])
			}
		}
	}
}

func TestEstimatorResources(t *testing.T) {
	sample1, sample2 := testIHPAGeneratorSample(t)
	tests := []struct {
		base     *ihpaGeneratorImpl
		expected []ihpav1beta2.Estimator
	}{
		{
			base: sample1,
			expected: []ihpav1beta2.Estimator{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ihpa-sample1-cpu",
						Namespace: "default",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion:         "ihpa.ake.cyberagent.co.jp/v1beta1",
								Controller:         func(b bool) *bool { return &b }(true),
								BlockOwnerDeletion: func(b bool) *bool { return &b }(true),
								Kind:               "IntelligentHorizontalPodAutoscaler.ihpa.ake.cyberagent.co.jp",
								Name:               "sample1",
								UID:                "9fc642f3-bb9d-404d-afd5-d04f1d6149ad",
							},
						},
						Annotations: map[string]string{
							"ihpa.ake.cyberagent.co.jp/fittingjob-id": "b9aa658a6b30a452ee08d0fbea2c4f40",
						},
					},
					Spec: ihpav1beta2.EstimatorSpec{
						Mode:       "adjust",
						GapMinutes: 5,
						MetricName: "ake.ihpa.forecasted_kubernetes_cpu_usage_total",
						MetricTags: []string{
							"kube_deployment:nginx",
							"kube_namespace:default",
							"kube_system_uid:46f9e396-d3c4-4103-a807-49054f47bbfb",
						},
						BaseMetricName: "kubernetes.cpu.usage.total",
						BaseMetricTags: []string{
							"kube_deployment:nginx",
							"kube_namespace:default",
							"kube_system_uid:46f9e396-d3c4-4103-a807-49054f47bbfb",
						},
						DataConfigMap: corev1.LocalObjectReference{Name: "ihpa-sample1-cpu"},
						Provider: ihpav1beta2.MetricProvider{
							Name: "sample-provider",
							ProviderSource: ihpav1beta2.ProviderSource{
								Datadog: &ihpav1beta2.DatadogProviderSource{
									APIKey: "xxx",
									APPKey: "yyy",
								},
							},
						},
					},
				},
			},
		},
		{
			base: sample2,
			expected: []ihpav1beta2.Estimator{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ihpa-sample2-cpu",
						Namespace: "web",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion:         "ihpa.ake.cyberagent.co.jp/v1",
								Controller:         func(b bool) *bool { return &b }(true),
								BlockOwnerDeletion: func(b bool) *bool { return &b }(true),
								Kind:               "IntelligentHorizontalPodAutoscaler.ihpa.ake.cyberagent.co.jp",
								Name:               "sample2",
								UID:                "9fc642f3-bb9d-404d-afd5-d04f1d6149ad",
							},
						},
						Annotations: map[string]string{
							"ihpa.ake.cyberagent.co.jp/fittingjob-id": "69f9447bad75992b533334827ce092ad",
						},
					},
					Spec: ihpav1beta2.EstimatorSpec{
						Mode:       "raw",
						GapMinutes: 10,
						MetricName: "ake.ihpa.forecasted_kubernetes_cpu_usage_total",
						MetricTags: []string{
							"kube_namespace:web",
							"kube_statefulset:web-db",
							"kube_system_uid:9833e04f-a689-47f9-b588-36a616432abd",
						},
						BaseMetricName: "kubernetes.cpu.usage.total",
						BaseMetricTags: []string{
							"kube_namespace:web",
							"kube_statefulset:web-db",
							"kube_system_uid:9833e04f-a689-47f9-b588-36a616432abd",
						},
						DataConfigMap: corev1.LocalObjectReference{Name: "ihpa-sample2-cpu"},
						Provider: ihpav1beta2.MetricProvider{
							Name: "sample-provider",
							ProviderSource: ihpav1beta2.ProviderSource{
								Datadog: &ihpav1beta2.DatadogProviderSource{
									APIKey: "xxx",
									APPKey: "yyy",
								},
							},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ihpa-sample2-memory",
						Namespace: "web",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion:         "ihpa.ake.cyberagent.co.jp/v1",
								Controller:         func(b bool) *bool { return &b }(true),
								BlockOwnerDeletion: func(b bool) *bool { return &b }(true),
								Kind:               "IntelligentHorizontalPodAutoscaler.ihpa.ake.cyberagent.co.jp",
								Name:               "sample2",
								UID:                "9fc642f3-bb9d-404d-afd5-d04f1d6149ad",
							},
						},
						Annotations: map[string]string{
							"ihpa.ake.cyberagent.co.jp/fittingjob-id": "c5abea3be857f41dc43ed7dec2eee7e3",
						},
					},
					Spec: ihpav1beta2.EstimatorSpec{
						Mode:       "raw",
						GapMinutes: 10,
						MetricName: "ake.ihpa.forecasted_kubernetes_memory_usage",
						MetricTags: []string{
							"kube_namespace:web",
							"kube_statefulset:web-db",
							"kube_system_uid:9833e04f-a689-47f9-b588-36a616432abd",
						},
						BaseMetricName: "kubernetes.memory.usage",
						BaseMetricTags: []string{
							"kube_namespace:web",
							"kube_statefulset:web-db",
							"kube_system_uid:9833e04f-a689-47f9-b588-36a616432abd",
						},
						DataConfigMap: corev1.LocalObjectReference{Name: "ihpa-sample2-memory"},
						Provider: ihpav1beta2.MetricProvider{
							Name: "sample-provider",
							ProviderSource: ihpav1beta2.ProviderSource{
								Datadog: &ihpav1beta2.DatadogProviderSource{
									APIKey: "xxx",
									APPKey: "yyy",
								},
							},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ihpa-sample2-nginx-net-request-per-s",
						Namespace: "web",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion:         "ihpa.ake.cyberagent.co.jp/v1",
								Controller:         func(b bool) *bool { return &b }(true),
								BlockOwnerDeletion: func(b bool) *bool { return &b }(true),
								Kind:               "IntelligentHorizontalPodAutoscaler.ihpa.ake.cyberagent.co.jp",
								Name:               "sample2",
								UID:                "9fc642f3-bb9d-404d-afd5-d04f1d6149ad",
							},
						},
						Annotations: map[string]string{
							"ihpa.ake.cyberagent.co.jp/fittingjob-id": "e9932760a233dffd256e6e0de941096e",
						},
					},
					Spec: ihpav1beta2.EstimatorSpec{
						Mode:       "raw",
						GapMinutes: 10,
						MetricName: "ake.ihpa.forecasted_nginx_net_request_per_s",
						MetricTags: []string{
							"kube_namespace:web",
							"kube_statefulset:web-db",
							"kube_system_uid:9833e04f-a689-47f9-b588-36a616432abd",
						},
						BaseMetricName: "nginx.net.request_per_s",
						BaseMetricTags: []string{
							"kube_app:sample",
							"kube_namespace:web",
							"kube_statefulset:web-db",
						},
						DataConfigMap: corev1.LocalObjectReference{Name: "ihpa-sample2-nginx-net-request-per-s"},
						Provider: ihpav1beta2.MetricProvider{
							Name: "sample-provider",
							ProviderSource: ihpav1beta2.ProviderSource{
								Datadog: &ihpav1beta2.DatadogProviderSource{
									APIKey: "xxx",
									APPKey: "yyy",
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		got, err := tt.base.EstimatorResources()
		if err != nil {
			t.Fatal(err)
		}
		for i := range got {
			if !reflect.DeepEqual(*(got[i]), tt.expected[i]) {
				t.Fatalf("estimator resources are not match\ngot=%#v\nexp=%#v", *(got[i]), tt.expected[i])
			}
		}
	}
}

func TestRBACResources(t *testing.T) {
	sample1, sample2 := testIHPAGeneratorSample(t)
	tests := []struct {
		generator           *ihpaGeneratorImpl
		expectedSA          *corev1.ServiceAccount
		expectedRole        *rbacv1.Role
		expectedRoleBinding *rbacv1.RoleBinding
	}{
		{
			generator: sample1,
			expectedSA: &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ihpa-sample1",
					Namespace: "default",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         "ihpa.ake.cyberagent.co.jp/v1beta1",
							Controller:         func(b bool) *bool { return &b }(true),
							BlockOwnerDeletion: func(b bool) *bool { return &b }(true),
							Kind:               "IntelligentHorizontalPodAutoscaler.ihpa.ake.cyberagent.co.jp",
							Name:               "sample1",
							UID:                "9fc642f3-bb9d-404d-afd5-d04f1d6149ad",
						},
					},
				},
			},
			expectedRole: &rbacv1.Role{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ihpa-sample1",
					Namespace: "default",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         "ihpa.ake.cyberagent.co.jp/v1beta1",
							Controller:         func(b bool) *bool { return &b }(true),
							BlockOwnerDeletion: func(b bool) *bool { return &b }(true),
							Kind:               "IntelligentHorizontalPodAutoscaler.ihpa.ake.cyberagent.co.jp",
							Name:               "sample1",
							UID:                "9fc642f3-bb9d-404d-afd5-d04f1d6149ad",
						},
					},
				},
				Rules: []rbacv1.PolicyRule{
					{
						APIGroups:     []string{""},
						Resources:     []string{"configmaps"},
						ResourceNames: []string{"ihpa-sample1-cpu"},
						Verbs:         []string{"get", "update", "patch"},
					},
				},
			},
			expectedRoleBinding: &rbacv1.RoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ihpa-sample1",
					Namespace: "default",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         "ihpa.ake.cyberagent.co.jp/v1beta1",
							Controller:         func(b bool) *bool { return &b }(true),
							BlockOwnerDeletion: func(b bool) *bool { return &b }(true),
							Kind:               "IntelligentHorizontalPodAutoscaler.ihpa.ake.cyberagent.co.jp",
							Name:               "sample1",
							UID:                "9fc642f3-bb9d-404d-afd5-d04f1d6149ad",
						},
					},
				},
				RoleRef: rbacv1.RoleRef{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "Role",
					Name:     "ihpa-sample1",
				},
				Subjects: []rbacv1.Subject{
					{
						APIGroup:  "",
						Kind:      "ServiceAccount",
						Name:      "ihpa-sample1",
						Namespace: "default",
					},
				},
			},
		},
		{
			generator: sample2,
			expectedSA: &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ihpa-sample2",
					Namespace: "web",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         "ihpa.ake.cyberagent.co.jp/v1",
							Controller:         func(b bool) *bool { return &b }(true),
							BlockOwnerDeletion: func(b bool) *bool { return &b }(true),
							Kind:               "IntelligentHorizontalPodAutoscaler.ihpa.ake.cyberagent.co.jp",
							Name:               "sample2",
							UID:                "9fc642f3-bb9d-404d-afd5-d04f1d6149ad",
						},
					},
				},
			},
			expectedRole: &rbacv1.Role{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ihpa-sample2",
					Namespace: "web",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         "ihpa.ake.cyberagent.co.jp/v1",
							Controller:         func(b bool) *bool { return &b }(true),
							BlockOwnerDeletion: func(b bool) *bool { return &b }(true),
							Kind:               "IntelligentHorizontalPodAutoscaler.ihpa.ake.cyberagent.co.jp",
							Name:               "sample2",
							UID:                "9fc642f3-bb9d-404d-afd5-d04f1d6149ad",
						},
					},
				},
				Rules: []rbacv1.PolicyRule{
					{
						APIGroups: []string{""},
						Resources: []string{"configmaps"},
						ResourceNames: []string{
							"ihpa-sample2-cpu",
							"ihpa-sample2-memory",
							"ihpa-sample2-nginx-net-request-per-s",
						},
						Verbs: []string{"get", "update", "patch"},
					},
				},
			},
			expectedRoleBinding: &rbacv1.RoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ihpa-sample2",
					Namespace: "web",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         "ihpa.ake.cyberagent.co.jp/v1",
							Controller:         func(b bool) *bool { return &b }(true),
							BlockOwnerDeletion: func(b bool) *bool { return &b }(true),
							Kind:               "IntelligentHorizontalPodAutoscaler.ihpa.ake.cyberagent.co.jp",
							Name:               "sample2",
							UID:                "9fc642f3-bb9d-404d-afd5-d04f1d6149ad",
						},
					},
				},
				RoleRef: rbacv1.RoleRef{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "Role",
					Name:     "ihpa-sample2",
				},
				Subjects: []rbacv1.Subject{
					{
						APIGroup:  "",
						Kind:      "ServiceAccount",
						Name:      "ihpa-sample2",
						Namespace: "web",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		gotSA, gotRole, gotRoleBinding, err := tt.generator.RBACResources()
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(gotSA, tt.expectedSA) {
			t.Fatalf("serviceAccount is not match (got=%#v, exp=%#v)", gotSA, tt.expectedSA)
		}
		if !reflect.DeepEqual(gotRole, tt.expectedRole) {
			t.Fatalf("role is not match (got=%#v, exp=%#v)", gotRole, tt.expectedRole)
		}
		if !reflect.DeepEqual(gotRoleBinding, tt.expectedRoleBinding) {
			t.Fatalf("roleBinding is not match (got=%#v, exp=%#v)", gotRoleBinding, tt.expectedRoleBinding)
		}
	}
}

func TestGenerateForecastedMetricSpec(t *testing.T) {
	sample1, sample2 := testIHPAGeneratorSample(t)
	tests := []struct {
		generator *ihpaGeneratorImpl
		metric    *autoscalingv2beta2.MetricSpec
		expected  *autoscalingv2beta2.MetricSpec
	}{
		{
			generator: sample1,
			metric: &autoscalingv2beta2.MetricSpec{
				Type: "Resource",
				Resource: &autoscalingv2beta2.ResourceMetricSource{
					Name: "cpu",
					Target: autoscalingv2beta2.MetricTarget{
						Type:               "Utilization",
						AverageUtilization: func(i int32) *int32 { return &i }(30),
					},
				},
			},
			expected: &autoscalingv2beta2.MetricSpec{
				Type: "External",
				External: &autoscalingv2beta2.ExternalMetricSource{
					Metric: autoscalingv2beta2.MetricIdentifier{
						Name: "ake.ihpa.forecasted_kubernetes_cpu_usage_total",
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"kube_system_uid": "46f9e396-d3c4-4103-a807-49054f47bbfb",
								"kube_namespace":  "default",
								"kube_deployment": "nginx",
							},
						},
					},
					Target: autoscalingv2beta2.MetricTarget{
						Type:         "AverageValue",
						AverageValue: resource.NewQuantity(resource.NewQuantity(150, resource.DecimalSI).ScaledValue(resource.Micro), resource.DecimalSI),
					},
				},
			},
		},
		{
			generator: sample2,
			metric: &autoscalingv2beta2.MetricSpec{
				Type: "External",
				External: &autoscalingv2beta2.ExternalMetricSource{
					Metric: autoscalingv2beta2.MetricIdentifier{
						Name: "nginx.net.request_per_s",
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"ownlabel": "hello",
							},
						},
					},
					Target: autoscalingv2beta2.MetricTarget{
						Type:         "AverageValue",
						AverageValue: resource.NewQuantity(100, resource.DecimalSI),
					},
				},
			},
			expected: &autoscalingv2beta2.MetricSpec{
				Type: "External",
				External: &autoscalingv2beta2.ExternalMetricSource{
					Metric: autoscalingv2beta2.MetricIdentifier{
						Name: "ake.ihpa.forecasted_nginx_net_request_per_s",
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"kube_system_uid":  "9833e04f-a689-47f9-b588-36a616432abd",
								"kube_namespace":   "web",
								"kube_statefulset": "web-db",
							},
						},
					},
					Target: autoscalingv2beta2.MetricTarget{
						Type:         "AverageValue",
						AverageValue: resource.NewQuantity(100, resource.DecimalSI),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		got, err := tt.generator.generateForecastedMetricSpec(tt.metric)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, tt.expected) {
			t.Fatalf("metric identifier is not match (got=%s, exp=%s)", got, tt.expected)
		}
	}
}

func TestConvertMetricSpecToIdentifier(t *testing.T) {
	sample1, sample2 := testIHPAGeneratorSample(t)
	tests := []struct {
		generator *ihpaGeneratorImpl
		metric    *autoscalingv2beta2.MetricSpec
		expected  *autoscalingv2beta2.MetricIdentifier
	}{
		{
			generator: sample1,
			metric: &autoscalingv2beta2.MetricSpec{
				Type: "Resource",
				Resource: &autoscalingv2beta2.ResourceMetricSource{
					Name: "cpu",
					Target: autoscalingv2beta2.MetricTarget{
						Type:               "Utilization",
						AverageUtilization: func(i int32) *int32 { return &i }(30),
					},
				},
			},
			expected: &autoscalingv2beta2.MetricIdentifier{
				Name: "kubernetes.cpu.usage.total",
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"kube_system_uid": "46f9e396-d3c4-4103-a807-49054f47bbfb",
						"kube_namespace":  "default",
						"kube_deployment": "nginx",
					},
				},
			},
		},
		{
			generator: sample2,
			metric: &autoscalingv2beta2.MetricSpec{
				Type: "External",
				External: &autoscalingv2beta2.ExternalMetricSource{
					Metric: autoscalingv2beta2.MetricIdentifier{
						Name: "nginx.net.request_per_s",
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"super_identifier": "hi",
							},
						},
					},
					Target: autoscalingv2beta2.MetricTarget{
						Type:         "AverageValue",
						AverageValue: resource.NewQuantity(100, resource.DecimalSI),
					},
				},
			},
			expected: &autoscalingv2beta2.MetricIdentifier{
				Name: "nginx.net.request_per_s",
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"super_identifier": "hi",
					},
				},
			},
		},
		{
			generator: sample2,
			metric: &autoscalingv2beta2.MetricSpec{
				Type: "Object",
				Object: &autoscalingv2beta2.ObjectMetricSource{
					Metric: autoscalingv2beta2.MetricIdentifier{
						Name: "requests-per-second",
					},
					DescribedObject: autoscalingv2beta2.CrossVersionObjectReference{
						Kind:       "Ingress",
						Name:       "main-route",
						APIVersion: "networking.k8s.io/v1beta1",
					},
					Target: autoscalingv2beta2.MetricTarget{
						Type:  "Value",
						Value: resource.NewQuantity(2000, resource.DecimalSI),
					},
				},
			},
			expected: nil,
		},
	}

	for _, tt := range tests {
		got, err := tt.generator.convertMetricSpecToIdentifier(tt.metric)
		if err != nil {
			if tt.expected == nil {
				continue
			}
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, tt.expected) {
			t.Fatalf("metric identifier is not match (got=%s, exp=%s)", got, tt.expected)
		}
	}
}

func TestUniqueMetricID(t *testing.T) {
	sample1, sample2 := testIHPAGeneratorSample(t)
	tests := []struct {
		generator *ihpaGeneratorImpl
		metric    *autoscalingv2beta2.MetricSpec
		expected  string
	}{
		{
			generator: sample1,
			metric: &autoscalingv2beta2.MetricSpec{
				Type: "Resource",
				Resource: &autoscalingv2beta2.ResourceMetricSource{
					Name: "cpu",
					Target: autoscalingv2beta2.MetricTarget{
						Type:               "Utilization",
						AverageUtilization: func(i int32) *int32 { return &i }(30),
					},
				},
			},
			expected: "b9aa658a6b30a452ee08d0fbea2c4f40",
		},
		{
			generator: sample1,
			metric: &autoscalingv2beta2.MetricSpec{
				Type: "External",
				External: &autoscalingv2beta2.ExternalMetricSource{
					Metric: autoscalingv2beta2.MetricIdentifier{
						Name: "hello_world.metric",
					},
					Target: autoscalingv2beta2.MetricTarget{
						Type:         "AverageValue",
						AverageValue: resource.NewQuantity(100, resource.DecimalSI),
					},
				},
			},
			expected: "3f4d9eb4d50045a484a41fb3ba466dd6",
		},
		{
			generator: sample2,
			metric: &autoscalingv2beta2.MetricSpec{
				Type: "Resource",
				Resource: &autoscalingv2beta2.ResourceMetricSource{
					Name: "memory",
					Target: autoscalingv2beta2.MetricTarget{
						Type:               "Utilization",
						AverageUtilization: func(i int32) *int32 { return &i }(30),
					},
				},
			},
			expected: "c5abea3be857f41dc43ed7dec2eee7e3",
		},
	}

	for _, tt := range tests {
		got := tt.generator.uniqueMetricID(tt.metric)
		if got != tt.expected {
			t.Fatalf("metric id is not match (got=%s, exp=%s)", got, tt.expected)
		}
	}
}

func TestUniqueMetricTags(t *testing.T) {
	sample1, sample2 := testIHPAGeneratorSample(t)
	tests := []struct {
		generator *ihpaGeneratorImpl
		expected  []string
	}{
		{
			generator: sample1,
			expected: []string{
				"kube_system_uid:46f9e396-d3c4-4103-a807-49054f47bbfb",
				"kube_namespace:default",
				"kube_deployment:nginx",
			},
		},
		{
			generator: sample2,
			expected: []string{
				"kube_system_uid:9833e04f-a689-47f9-b588-36a616432abd",
				"kube_namespace:web",
				"kube_statefulset:web-db",
			},
		},
	}

	for _, tt := range tests {
		got := tt.generator.uniqueMetricTags()
		if len(got) != len(tt.expected) {
			t.Fatalf("tags is not match (got=%#v, exp=%#v)", got, tt.expected)
		}
		exists := make(map[string]struct{})
		for _, v := range tt.expected {
			exists[v] = struct{}{}
		}
		for _, v := range got {
			if _, ok := exists[v]; !ok {
				t.Fatalf("tags is not match (got=%#v, exp=%#v)", got, tt.expected)
			}
		}
	}
}

func TestUniqueMetricFilters(t *testing.T) {
	sample1, sample2 := testIHPAGeneratorSample(t)
	tests := []struct {
		generator *ihpaGeneratorImpl
		expected  map[string]string
	}{
		{
			generator: sample1,
			expected: map[string]string{
				"kube_system_uid": "46f9e396-d3c4-4103-a807-49054f47bbfb",
				"kube_namespace":  "default",
				"kube_deployment": "nginx",
			},
		},
		{
			generator: sample2,
			expected: map[string]string{
				"kube_system_uid":  "9833e04f-a689-47f9-b588-36a616432abd",
				"kube_namespace":   "web",
				"kube_statefulset": "web-db",
			},
		},
	}

	for _, tt := range tests {
		got := tt.generator.uniqueMetricFilters()
		if !reflect.DeepEqual(got, tt.expected) {
			t.Fatalf("filter map is not match (got=%v, exp=%v)", got, tt.expected)
		}
	}
}

func TestIhpaMetricString(t *testing.T) {
	sample1, sample2 := testIHPAGeneratorSample(t)
	tests := []struct {
		generator *ihpaGeneratorImpl
		metric    *autoscalingv2beta2.MetricSpec
		expected  string
	}{
		{
			generator: sample1,
			metric: &autoscalingv2beta2.MetricSpec{
				Type: "Resource",
				Resource: &autoscalingv2beta2.ResourceMetricSource{
					Name: "cpu",
					Target: autoscalingv2beta2.MetricTarget{
						Type:               "Utilization",
						AverageUtilization: func(i int32) *int32 { return &i }(30),
					},
				},
			},
			expected: "ihpa-sample1-cpu",
		},
		{
			generator: sample1,
			metric: &autoscalingv2beta2.MetricSpec{
				Type: "External",
				Resource: &autoscalingv2beta2.ResourceMetricSource{
					Name: "cpu",
					Target: autoscalingv2beta2.MetricTarget{
						Type:               "Utilization",
						AverageUtilization: func(i int32) *int32 { return &i }(30),
					},
				},
				External: &autoscalingv2beta2.ExternalMetricSource{
					Metric: autoscalingv2beta2.MetricIdentifier{
						Name: "hello_world.metric",
					},
					Target: autoscalingv2beta2.MetricTarget{
						Type:         "AverageValue",
						AverageValue: resource.NewQuantity(100, resource.DecimalSI),
					},
				},
			},
			expected: "ihpa-sample1-hello-world-metric",
		},
		{
			generator: sample2,
			metric: &autoscalingv2beta2.MetricSpec{
				Type: "External",
				External: &autoscalingv2beta2.ExternalMetricSource{
					Metric: autoscalingv2beta2.MetricIdentifier{
						Name: "nginx.net.request_per_s",
					},
					Target: autoscalingv2beta2.MetricTarget{
						Type:         "AverageValue",
						AverageValue: resource.NewQuantity(100, resource.DecimalSI),
					},
				},
			},
			expected: "ihpa-sample2-nginx-net-request-per-s",
		},
	}

	for _, tt := range tests {
		got := tt.generator.ihpaMetricString(tt.metric)
		if got != tt.expected {
			t.Fatalf("ihpaMetricString is not match (got=%s, exp=%s)", got, tt.expected)
		}
	}
}

func TestIhpaString(t *testing.T) {
	sample1, sample2 := testIHPAGeneratorSample(t)
	tests := []struct {
		generator *ihpaGeneratorImpl
		expected  string
	}{
		{
			generator: sample1,
			expected:  "ihpa-sample1",
		},
		{
			generator: sample2,
			expected:  "ihpa-sample2",
		},
	}

	for _, tt := range tests {
		got := tt.generator.ihpaString()
		if got != tt.expected {
			t.Fatalf("ihpaString is not match (got=%s, exp=%s)", got, tt.expected)
		}
	}
}
