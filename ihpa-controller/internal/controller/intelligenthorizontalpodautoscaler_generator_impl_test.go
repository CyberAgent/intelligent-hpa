package controller

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	ihpav1beta2 "github.com/cyberagent-oss/intelligent-hpa/ihpa-controller/api/v1beta2"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func testIHPAGeneratorSample(t *testing.T) (*ihpaGeneratorImpl, *ihpaGeneratorImpl) {
	const trueValue = "true"
	const yyyValue3 = "yyy"
	const xxxValue3 = "xxx"
	const webDbValue2 = "web-db"
	const webValue6 = "web"
	const secret1Value2 = "secret1"
	const sample1Value4 = "sample1"
	const sampleProviderValue2 = "sample-provider"
	const sampleValue = "sample"
	const nginxNetRequestPerSValue5 = "nginx.net.request_per_s"
	const nginxValue4 = "nginx"
	const memoryValue = "memory"
	const kubeStatefulsetValue3 = "kube_statefulset"
	const kubeNamespaceValue4 = "kube_namespace"
	const kubeApp = "kube_app"
	const ihpaAkeCyberagentCoJpV1beta1Value3 = "ihpa.ake.cyberagent.co.jp/v1beta1"
	const ihpaAkeCyberagentCoJpV1Value3 = "ihpa.ake.cyberagent.co.jp/v1"
	const defaultValue8 = "default"
	const cpuValue5 = "cpu"
	const appsV1 = "apps/v1"
	const adjustValue = "adjust"
	const utilizationValue4 = "Utilization"
	const statefulSetValue = "StatefulSet"
	const resourceValue5 = "Resource"
	const intelligentHorizontalPodAutoscaler = "IntelligentHorizontalPodAutoscaler"
	const externalValue4 = "External"
	const deploymentValue = "Deployment"
	const averageValueValue5 = "AverageValue"
	const id9fc642f3Bb9d404dAfd5D04f1d6149adValue4 = "9fc642f3-bb9d-404d-afd5-d04f1d6149ad"
	const id9833e04fA68947f9B58836a616432abdValue2 = "9833e04f-a689-47f9-b588-36a616432abd"
	const id46f9e396D3c44103A80749054f47bbfbValue3 = "46f9e396-d3c4-4103-a807-49054f47bbfb"
	t.Helper()
	sample1 := &ihpaGeneratorImpl{
		kubeSystemUID: id46f9e396D3c44103A80749054f47bbfbValue3,
		scaleTargetRequests: corev1.ResourceList{
			corev1.ResourceCPU: *resource.NewScaledQuantity(500, resource.Milli),
		},
		ihpa: &ihpav1beta2.IntelligentHorizontalPodAutoscaler{
			ObjectMeta: metav1.ObjectMeta{
				Name:      sample1Value4,
				Namespace: defaultValue8,
				UID:       id9fc642f3Bb9d404dAfd5D04f1d6149adValue4,
			},
			TypeMeta: metav1.TypeMeta{
				Kind:       intelligentHorizontalPodAutoscaler,
				APIVersion: ihpaAkeCyberagentCoJpV1beta1Value3,
			},
			Spec: ihpav1beta2.IntelligentHorizontalPodAutoscalerSpec{
				HorizontalPodAutoscalerTemplate: ihpav1beta2.ExtendedHorizontalPodAutoscalerTemplateSpec{
					Spec: ihpav1beta2.ExtendedHorizontalPodAutoscalerSpec{
						ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
							APIVersion: appsV1,
							Kind:       deploymentValue,
							Name:       nginxValue4,
						},
						MinReplicas: func(i int32) *int32 { return &i }(3),
						Metrics: []ihpav1beta2.ExtendedMetricSpec{
							{
								Type: resourceValue5,
								Resource: &autoscalingv2.ResourceMetricSource{
									Name: cpuValue5,
									Target: autoscalingv2.MetricTarget{
										Type:               utilizationValue4,
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
					Mode:       adjustValue,
					GapMinutes: 5,
				},
				MetricProvider: ihpav1beta2.MetricProvider{
					Name: sampleProviderValue2,
					ProviderSource: ihpav1beta2.ProviderSource{
						Datadog: &ihpav1beta2.DatadogProviderSource{
							APIKey: xxxValue3,
							APPKey: yyyValue3,
						},
					},
				},
			},
		},
	}
	sample2 := &ihpaGeneratorImpl{
		kubeSystemUID: id9833e04fA68947f9B58836a616432abdValue2,
		scaleTargetRequests: corev1.ResourceList{
			corev1.ResourceCPU:     *resource.NewScaledQuantity(2, 0),
			corev1.ResourceMemory:  *resource.NewScaledQuantity(500, resource.Mega),
			corev1.ResourceStorage: *resource.NewScaledQuantity(4, resource.Giga),
		},
		ihpa: &ihpav1beta2.IntelligentHorizontalPodAutoscaler{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "sample2",
				Namespace: webValue6,
				UID:       id9fc642f3Bb9d404dAfd5D04f1d6149adValue4,
			},
			TypeMeta: metav1.TypeMeta{
				Kind:       intelligentHorizontalPodAutoscaler,
				APIVersion: ihpaAkeCyberagentCoJpV1Value3,
			},
			Spec: ihpav1beta2.IntelligentHorizontalPodAutoscalerSpec{
				HorizontalPodAutoscalerTemplate: ihpav1beta2.ExtendedHorizontalPodAutoscalerTemplateSpec{
					Spec: ihpav1beta2.ExtendedHorizontalPodAutoscalerSpec{
						ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
							APIVersion: appsV1,
							Kind:       statefulSetValue,
							Name:       webDbValue2,
						},
						MinReplicas: func(i int32) *int32 { return &i }(5),
						MaxReplicas: 10,
						Metrics: []ihpav1beta2.ExtendedMetricSpec{
							{
								Type: resourceValue5,
								Resource: &autoscalingv2.ResourceMetricSource{
									Name: cpuValue5,
									Target: autoscalingv2.MetricTarget{
										Type:               utilizationValue4,
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
											{Name: secret1Value2},
											{Name: "secret2"},
										},
										Image: "my_image:v1",
									},
								},
							},
							{
								Type: resourceValue5,
								Resource: &autoscalingv2.ResourceMetricSource{
									Name: memoryValue,
									Target: autoscalingv2.MetricTarget{
										Type:               utilizationValue4,
										AverageUtilization: func(i int32) *int32 { return &i }(80),
									},
								},
								// omit FittingJobPatch
							},
							{
								Type: externalValue4,
								External: &autoscalingv2.ExternalMetricSource{
									Metric: autoscalingv2.MetricIdentifier{
										Name: nginxNetRequestPerSValue5,
										Selector: &metav1.LabelSelector{
											MatchLabels: map[string]string{
												kubeStatefulsetValue3: webDbValue2,
												kubeNamespaceValue4:   webValue6,
												kubeApp:               sampleValue,
											},
										},
									},
									Target: autoscalingv2.MetricTarget{
										Type:         averageValueValue5,
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
										ImagePullSecrets: []corev1.LocalObjectReference{{Name: secret1Value2}},
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
										NodeSelector: map[string]string{"gpu": trueValue},
										Tolerations: []corev1.Toleration{
											{
												Key:      "master",
												Operator: "Equal",
												Value:    trueValue,
												Effect:   "NoSchedule",
											},
										},
										ServiceAccountName: "sa1",
										Volumes: []corev1.Volume{
											{
												Name: "volume1",
												VolumeSource: corev1.VolumeSource{
													Secret: &corev1.SecretVolumeSource{SecretName: secret1Value2},
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
					Name: sampleProviderValue2,
					ProviderSource: ihpav1beta2.ProviderSource{
						Datadog: &ihpav1beta2.DatadogProviderSource{
							APIKey: xxxValue3,
							APPKey: yyyValue3,
						},
					},
				},
			},
		},
	}
	return sample1, sample2
}

func TestHorizontalPodAutoscalerResource(t *testing.T) {
	const webDbValue = "web-db"
	const webValue5 = "web"
	const sample1Value3 = "sample1"
	const nginxNetRequestPerSValue4 = "nginx.net.request_per_s"
	const nginxValue3 = "nginx"
	const kubeSystemUidValue3 = "kube_system_uid"
	const kubeStatefulsetValue2 = "kube_statefulset"
	const kubeNamespaceValue3 = "kube_namespace"
	const kubeDeploymentValue3 = "kube_deployment"
	const ihpaAkeCyberagentCoJpV1beta1Value2 = "ihpa.ake.cyberagent.co.jp/v1beta1"
	const ihpaSample2Value2 = "ihpa-sample2"
	const ihpaSample1Value2 = "ihpa-sample1"
	const defaultValue7 = "default"
	const cpuValue4 = "cpu"
	const akeIhpaForecastedNginxNetRequestPerS = "ake.ihpa.forecasted_nginx_net_request_per_s"
	const akeIhpaForecastedKubernetesCpuUsageTotalValue = "ake.ihpa.forecasted_kubernetes_cpu_usage_total"
	const utilizationValue3 = "Utilization"
	const statefulSet = "StatefulSet"
	const resourceValue4 = "Resource"
	const intelligentHorizontalPodAutoscalerIhpaAkeCyberagentCoJpValue4 = "IntelligentHorizontalPodAutoscaler.ihpa.ake.cyberagent.co.jp"
	const externalValue3 = "External"
	const deployment = "Deployment"
	const averageValueValue4 = "AverageValue"
	const id9fc642f3Bb9d404dAfd5D04f1d6149adValue3 = "9fc642f3-bb9d-404d-afd5-d04f1d6149ad"
	const id9833e04fA68947f9B58836a616432abdValue = "9833e04f-a689-47f9-b588-36a616432abd"
	const id46f9e396D3c44103A80749054f47bbfbValue2 = "46f9e396-d3c4-4103-a807-49054f47bbfb"
	sample1, sample2 := testIHPAGeneratorSample(t)
	tests := []struct {
		base     *ihpaGeneratorImpl
		expected *autoscalingv2.HorizontalPodAutoscaler
	}{
		{
			base: sample1,
			expected: &autoscalingv2.HorizontalPodAutoscaler{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ihpaSample1Value2,
					Namespace: defaultValue7,
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         ihpaAkeCyberagentCoJpV1beta1Value2,
							Controller:         func(b bool) *bool { return &b }(true),
							BlockOwnerDeletion: func(b bool) *bool { return &b }(true),
							Kind:               intelligentHorizontalPodAutoscalerIhpaAkeCyberagentCoJpValue4,
							Name:               sample1Value3,
							UID:                id9fc642f3Bb9d404dAfd5D04f1d6149adValue3,
						},
					},
				},
				Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
					ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
						APIVersion: "apps/v1",
						Kind:       deployment,
						Name:       nginxValue3,
					},
					MinReplicas: func(i int32) *int32 { return &i }(3),
					Metrics: []autoscalingv2.MetricSpec{
						{
							Type: resourceValue4,
							Resource: &autoscalingv2.ResourceMetricSource{
								Name: cpuValue4,
								Target: autoscalingv2.MetricTarget{
									Type:               utilizationValue3,
									AverageUtilization: func(i int32) *int32 { return &i }(50),
								},
							},
						},
						{
							Type: externalValue3,
							External: &autoscalingv2.ExternalMetricSource{
								Metric: autoscalingv2.MetricIdentifier{
									Name: akeIhpaForecastedKubernetesCpuUsageTotalValue,
									Selector: &metav1.LabelSelector{
										MatchLabels: map[string]string{
											kubeSystemUidValue3:  id46f9e396D3c44103A80749054f47bbfbValue2,
											kubeDeploymentValue3: nginxValue3,
											kubeNamespaceValue3:  defaultValue7,
										},
									},
								},
								Target: autoscalingv2.MetricTarget{
									Type:         averageValueValue4,
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
			expected: &autoscalingv2.HorizontalPodAutoscaler{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ihpaSample2Value2,
					Namespace: webValue5,
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         "ihpa.ake.cyberagent.co.jp/v1",
							Controller:         func(b bool) *bool { return &b }(true),
							BlockOwnerDeletion: func(b bool) *bool { return &b }(true),
							Kind:               intelligentHorizontalPodAutoscalerIhpaAkeCyberagentCoJpValue4,
							Name:               "sample2",
							UID:                id9fc642f3Bb9d404dAfd5D04f1d6149adValue3,
						},
					},
				},
				Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
					ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
						APIVersion: "apps/v1",
						Kind:       statefulSet,
						Name:       webDbValue,
					},
					MinReplicas: func(i int32) *int32 { return &i }(5),
					MaxReplicas: 10,
					Metrics: []autoscalingv2.MetricSpec{
						{
							Type: resourceValue4,
							Resource: &autoscalingv2.ResourceMetricSource{
								Name: cpuValue4,
								Target: autoscalingv2.MetricTarget{
									Type:               utilizationValue3,
									AverageUtilization: func(i int32) *int32 { return &i }(50),
								},
							},
						},
						{
							Type: resourceValue4,
							Resource: &autoscalingv2.ResourceMetricSource{
								Name: "memory",
								Target: autoscalingv2.MetricTarget{
									Type:               utilizationValue3,
									AverageUtilization: func(i int32) *int32 { return &i }(80),
								},
							},
						},
						{
							Type: externalValue3,
							External: &autoscalingv2.ExternalMetricSource{
								Metric: autoscalingv2.MetricIdentifier{
									Name: nginxNetRequestPerSValue4,
									Selector: &metav1.LabelSelector{
										MatchLabels: map[string]string{
											kubeStatefulsetValue2: webDbValue,
											kubeNamespaceValue3:   webValue5,
											"kube_app":            "sample",
										},
									},
								},
								Target: autoscalingv2.MetricTarget{
									Type:         averageValueValue4,
									AverageValue: resource.NewQuantity(50, resource.DecimalSI),
								},
							},
						},
						{
							Type: externalValue3,
							External: &autoscalingv2.ExternalMetricSource{
								Metric: autoscalingv2.MetricIdentifier{
									Name: akeIhpaForecastedKubernetesCpuUsageTotalValue,
									Selector: &metav1.LabelSelector{
										MatchLabels: map[string]string{
											kubeSystemUidValue3:   id9833e04fA68947f9B58836a616432abdValue,
											kubeStatefulsetValue2: webDbValue,
											kubeNamespaceValue3:   webValue5,
										},
									},
								},
								Target: autoscalingv2.MetricTarget{
									Type:         averageValueValue4,
									AverageValue: resource.NewQuantity(resource.NewQuantity(1000, resource.DecimalSI).ScaledValue(resource.Micro), resource.DecimalSI),
									// 1000M (1,000,000,000) (resource.Micro = 10^6)
								},
							},
						},
						{
							Type: externalValue3,
							External: &autoscalingv2.ExternalMetricSource{
								Metric: autoscalingv2.MetricIdentifier{
									Name: "ake.ihpa.forecasted_kubernetes_memory_usage",
									Selector: &metav1.LabelSelector{
										MatchLabels: map[string]string{
											kubeSystemUidValue3:   id9833e04fA68947f9B58836a616432abdValue,
											kubeStatefulsetValue2: webDbValue,
											kubeNamespaceValue3:   webValue5,
										},
									},
								},
								Target: autoscalingv2.MetricTarget{
									Type:         averageValueValue4,
									AverageValue: resource.NewQuantity(resource.NewQuantity(400_000_000, resource.DecimalSI).Value(), resource.DecimalSI),
								},
							},
						},
						{
							Type: externalValue3,
							External: &autoscalingv2.ExternalMetricSource{
								Metric: autoscalingv2.MetricIdentifier{
									Name: akeIhpaForecastedNginxNetRequestPerS,
									Selector: &metav1.LabelSelector{
										MatchLabels: map[string]string{
											kubeSystemUidValue3:   id9833e04fA68947f9B58836a616432abdValue,
											kubeStatefulsetValue2: webDbValue,
											kubeNamespaceValue3:   webValue5,
										},
									},
								},
								Target: autoscalingv2.MetricTarget{
									Type:         averageValueValue4,
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
			for i, target := range []*autoscalingv2.HorizontalPodAutoscaler{got, tt.expected} {
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
	const ihpaAkeCyberagentCoJpFittingjobIdValue = "ihpa.ake.cyberagent.co.jp/fittingjob-id"
	const yyyValue2 = "yyy"
	const xxxValue2 = "xxx"
	const webDb = "web-db"
	const webValue4 = "web"
	const secret1Value = "secret1"
	const sample2Value4 = "sample2"
	const sampleProviderValue = "sample-provider"
	const nginxNetRequestPerSValue3 = "nginx.net.request_per_s"
	const nginxValue2 = "nginx"
	const kubernetesCpuUsageTotalValue = "kubernetes.cpu.usage.total"
	const kubeSystemUidValue2 = "kube_system_uid"
	const kubeStatefulsetValue = "kube_statefulset"
	const kubeNamespaceValue2 = "kube_namespace"
	const kubeDeploymentValue2 = "kube_deployment"
	const ihpaAkeCyberagentCoJpV1beta1Value = "ihpa.ake.cyberagent.co.jp/v1beta1"
	const ihpaAkeCyberagentCoJpV1Value2 = "ihpa.ake.cyberagent.co.jp/v1"
	const ihpaSample2NginxNetRequestPerSValue = "ihpa-sample2-nginx-net-request-per-s"
	const ihpaSample2MemoryValue = "ihpa-sample2-memory"
	const ihpaSample2CpuValue = "ihpa-sample2-cpu"
	const ihpaSample2Value = "ihpa-sample2"
	const ihpaSample1CpuValue2 = "ihpa-sample1-cpu"
	const ihpaSample1Value = "ihpa-sample1"
	const defaultValue6 = "default"
	const c5abea3be857f41dc43ed7dec2eee7e3Value = "c5abea3be857f41dc43ed7dec2eee7e3"
	const b9aa658a6b30a452ee08d0fbea2c4f40Value = "b9aa658a6b30a452ee08d0fbea2c4f40"
	const intelligentHorizontalPodAutoscalerIhpaAkeCyberagentCoJpValue3 = "IntelligentHorizontalPodAutoscaler.ihpa.ake.cyberagent.co.jp"
	const id9fc642f3Bb9d404dAfd5D04f1d6149adValue2 = "9fc642f3-bb9d-404d-afd5-d04f1d6149ad"
	const id9833e04fA68947f9B58836a616432abd = "9833e04f-a689-47f9-b588-36a616432abd"
	const id46f9e396D3c44103A80749054f47bbfbValue = "46f9e396-d3c4-4103-a807-49054f47bbfb"
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
						Name:      ihpaSample1CpuValue2,
						Namespace: defaultValue6,
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion:         ihpaAkeCyberagentCoJpV1beta1Value,
								Controller:         func(b bool) *bool { return &b }(true),
								BlockOwnerDeletion: func(b bool) *bool { return &b }(true),
								Kind:               intelligentHorizontalPodAutoscalerIhpaAkeCyberagentCoJpValue3,
								Name:               "sample1",
								UID:                id9fc642f3Bb9d404dAfd5D04f1d6149adValue2,
							},
						},
						Annotations: map[string]string{
							ihpaAkeCyberagentCoJpFittingjobIdValue: b9aa658a6b30a452ee08d0fbea2c4f40Value,
						},
					},
					Spec: ihpav1beta2.FittingJobSpec{
						Seasonality:   "weekly",
						ExecuteOn:     4,
						DataConfigMap: corev1.LocalObjectReference{Name: ihpaSample1CpuValue2},
						JobPatchSpec: ihpav1beta2.JobPatchSpec{
							ServiceAccountName: ihpaSample1Value,
						},
						TargetMetric: autoscalingv2.MetricIdentifier{
							Name: kubernetesCpuUsageTotalValue,
							Selector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									kubeSystemUidValue2:  id46f9e396D3c44103A80749054f47bbfbValue,
									kubeDeploymentValue2: nginxValue2,
									kubeNamespaceValue2:  defaultValue6,
								},
							},
						},
						Provider: ihpav1beta2.MetricProvider{
							Name: sampleProviderValue,
							ProviderSource: ihpav1beta2.ProviderSource{
								Datadog: &ihpav1beta2.DatadogProviderSource{
									APIKey: xxxValue2,
									APPKey: yyyValue2,
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
						Name:      ihpaSample2CpuValue,
						Namespace: webValue4,
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion:         ihpaAkeCyberagentCoJpV1Value2,
								Controller:         func(b bool) *bool { return &b }(true),
								BlockOwnerDeletion: func(b bool) *bool { return &b }(true),
								Kind:               intelligentHorizontalPodAutoscalerIhpaAkeCyberagentCoJpValue3,
								Name:               sample2Value4,
								UID:                id9fc642f3Bb9d404dAfd5D04f1d6149adValue2,
							},
						},
						Annotations: map[string]string{
							ihpaAkeCyberagentCoJpFittingjobIdValue: "69f9447bad75992b533334827ce092ad",
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
						DataConfigMap: corev1.LocalObjectReference{Name: ihpaSample2CpuValue},
						JobPatchSpec: ihpav1beta2.JobPatchSpec{
							Image: "my_image:v1",
							ImagePullSecrets: []corev1.LocalObjectReference{
								{Name: secret1Value},
								{Name: "secret2"},
							},
							ServiceAccountName: ihpaSample2Value,
						},
						TargetMetric: autoscalingv2.MetricIdentifier{
							Name: kubernetesCpuUsageTotalValue,
							Selector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									kubeSystemUidValue2:  id9833e04fA68947f9B58836a616432abd,
									kubeStatefulsetValue: webDb,
									kubeNamespaceValue2:  webValue4,
								},
							},
						},
						Provider: ihpav1beta2.MetricProvider{
							Name: sampleProviderValue,
							ProviderSource: ihpav1beta2.ProviderSource{
								Datadog: &ihpav1beta2.DatadogProviderSource{
									APIKey: xxxValue2,
									APPKey: yyyValue2,
								},
							},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      ihpaSample2MemoryValue,
						Namespace: webValue4,
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion:         ihpaAkeCyberagentCoJpV1Value2,
								Controller:         func(b bool) *bool { return &b }(true),
								BlockOwnerDeletion: func(b bool) *bool { return &b }(true),
								Kind:               intelligentHorizontalPodAutoscalerIhpaAkeCyberagentCoJpValue3,
								Name:               sample2Value4,
								UID:                id9fc642f3Bb9d404dAfd5D04f1d6149adValue2,
							},
						},
						Annotations: map[string]string{
							ihpaAkeCyberagentCoJpFittingjobIdValue: c5abea3be857f41dc43ed7dec2eee7e3Value,
						},
					},
					Spec: ihpav1beta2.FittingJobSpec{
						DataConfigMap: corev1.LocalObjectReference{Name: ihpaSample2MemoryValue},
						JobPatchSpec: ihpav1beta2.JobPatchSpec{
							ServiceAccountName: ihpaSample2Value,
						},
						TargetMetric: autoscalingv2.MetricIdentifier{
							Name: "kubernetes.memory.usage",
							Selector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									kubeSystemUidValue2:  id9833e04fA68947f9B58836a616432abd,
									kubeStatefulsetValue: webDb,
									kubeNamespaceValue2:  webValue4,
								},
							},
						},
						Provider: ihpav1beta2.MetricProvider{
							Name: sampleProviderValue,
							ProviderSource: ihpav1beta2.ProviderSource{
								Datadog: &ihpav1beta2.DatadogProviderSource{
									APIKey: xxxValue2,
									APPKey: yyyValue2,
								},
							},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      ihpaSample2NginxNetRequestPerSValue,
						Namespace: webValue4,
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion:         ihpaAkeCyberagentCoJpV1Value2,
								Controller:         func(b bool) *bool { return &b }(true),
								BlockOwnerDeletion: func(b bool) *bool { return &b }(true),
								Kind:               intelligentHorizontalPodAutoscalerIhpaAkeCyberagentCoJpValue3,
								Name:               sample2Value4,
								UID:                id9fc642f3Bb9d404dAfd5D04f1d6149adValue2,
							},
						},
						Annotations: map[string]string{
							ihpaAkeCyberagentCoJpFittingjobIdValue: "e9932760a233dffd256e6e0de941096e",
						},
					},
					Spec: ihpav1beta2.FittingJobSpec{
						Seasonality:   "auto",
						ExecuteOn:     2,
						DataConfigMap: corev1.LocalObjectReference{Name: ihpaSample2NginxNetRequestPerSValue},
						JobPatchSpec: ihpav1beta2.JobPatchSpec{
							// related to Job
							ActiveDeadlineSeconds: func(i int64) *int64 { return &i }(300),
							BackoffLimit:          func(i int32) *int32 { return &i }(5),
							Completions:           func(i int32) *int32 { return &i }(10),
							// related to Pod
							ImagePullSecrets: []corev1.LocalObjectReference{{Name: secret1Value}},
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
										Secret: &corev1.SecretVolumeSource{SecretName: secret1Value},
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
						TargetMetric: autoscalingv2.MetricIdentifier{
							Name: nginxNetRequestPerSValue3,
							Selector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									kubeStatefulsetValue: webDb,
									kubeNamespaceValue2:  webValue4,
									"kube_app":           "sample",
								},
							},
						},
						Provider: ihpav1beta2.MetricProvider{
							Name: sampleProviderValue,
							ProviderSource: ihpav1beta2.ProviderSource{
								Datadog: &ihpav1beta2.DatadogProviderSource{
									APIKey: xxxValue2,
									APPKey: yyyValue2,
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
	const ihpaAkeCyberagentCoJpFittingjobId = "ihpa.ake.cyberagent.co.jp/fittingjob-id"
	const yyyValue = "yyy"
	const xxxValue = "xxx"
	const webValue3 = "web"
	const sample2Value3 = "sample2"
	const sampleProvider = "sample-provider"
	const rawValue = "raw"
	const nginxNetRequestPerSValue2 = "nginx.net.request_per_s"
	const kubernetesCpuUsageTotal = "kubernetes.cpu.usage.total"
	const kubeSystemUid9833e04fA68947f9B58836a616432abd = "kube_system_uid:9833e04f-a689-47f9-b588-36a616432abd"
	const kubeSystemUid46f9e396D3c44103A80749054f47bbfb = "kube_system_uid:46f9e396-d3c4-4103-a807-49054f47bbfb"
	const kubeStatefulsetWebDb = "kube_statefulset:web-db"
	const kubeNamespaceWeb = "kube_namespace:web"
	const kubeNamespaceDefault = "kube_namespace:default"
	const kubeDeploymentNginx = "kube_deployment:nginx"
	const ihpaAkeCyberagentCoJpV1Value = "ihpa.ake.cyberagent.co.jp/v1"
	const ihpaSample2NginxNetRequestPerS = "ihpa-sample2-nginx-net-request-per-s"
	const ihpaSample2Memory = "ihpa-sample2-memory"
	const ihpaSample2Cpu = "ihpa-sample2-cpu"
	const ihpaSample1CpuValue = "ihpa-sample1-cpu"
	const defaultValue5 = "default"
	const akeIhpaForecastedKubernetesCpuUsageTotal = "ake.ihpa.forecasted_kubernetes_cpu_usage_total"
	const intelligentHorizontalPodAutoscalerIhpaAkeCyberagentCoJpValue2 = "IntelligentHorizontalPodAutoscaler.ihpa.ake.cyberagent.co.jp"
	const id9fc642f3Bb9d404dAfd5D04f1d6149adValue = "9fc642f3-bb9d-404d-afd5-d04f1d6149ad"
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
						Name:      ihpaSample1CpuValue,
						Namespace: defaultValue5,
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion:         "ihpa.ake.cyberagent.co.jp/v1beta1",
								Controller:         func(b bool) *bool { return &b }(true),
								BlockOwnerDeletion: func(b bool) *bool { return &b }(true),
								Kind:               intelligentHorizontalPodAutoscalerIhpaAkeCyberagentCoJpValue2,
								Name:               "sample1",
								UID:                id9fc642f3Bb9d404dAfd5D04f1d6149adValue,
							},
						},
						Annotations: map[string]string{
							ihpaAkeCyberagentCoJpFittingjobId: "b9aa658a6b30a452ee08d0fbea2c4f40",
						},
					},
					Spec: ihpav1beta2.EstimatorSpec{
						Mode:       "adjust",
						GapMinutes: 5,
						MetricName: akeIhpaForecastedKubernetesCpuUsageTotal,
						MetricTags: []string{
							kubeDeploymentNginx,
							kubeNamespaceDefault,
							kubeSystemUid46f9e396D3c44103A80749054f47bbfb,
						},
						BaseMetricName: kubernetesCpuUsageTotal,
						BaseMetricTags: []string{
							kubeDeploymentNginx,
							kubeNamespaceDefault,
							kubeSystemUid46f9e396D3c44103A80749054f47bbfb,
						},
						DataConfigMap: corev1.LocalObjectReference{Name: ihpaSample1CpuValue},
						Provider: ihpav1beta2.MetricProvider{
							Name: sampleProvider,
							ProviderSource: ihpav1beta2.ProviderSource{
								Datadog: &ihpav1beta2.DatadogProviderSource{
									APIKey: xxxValue,
									APPKey: yyyValue,
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
						Name:      ihpaSample2Cpu,
						Namespace: webValue3,
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion:         ihpaAkeCyberagentCoJpV1Value,
								Controller:         func(b bool) *bool { return &b }(true),
								BlockOwnerDeletion: func(b bool) *bool { return &b }(true),
								Kind:               intelligentHorizontalPodAutoscalerIhpaAkeCyberagentCoJpValue2,
								Name:               sample2Value3,
								UID:                id9fc642f3Bb9d404dAfd5D04f1d6149adValue,
							},
						},
						Annotations: map[string]string{
							ihpaAkeCyberagentCoJpFittingjobId: "69f9447bad75992b533334827ce092ad",
						},
					},
					Spec: ihpav1beta2.EstimatorSpec{
						Mode:       rawValue,
						GapMinutes: 10,
						MetricName: akeIhpaForecastedKubernetesCpuUsageTotal,
						MetricTags: []string{
							kubeNamespaceWeb,
							kubeStatefulsetWebDb,
							kubeSystemUid9833e04fA68947f9B58836a616432abd,
						},
						BaseMetricName: kubernetesCpuUsageTotal,
						BaseMetricTags: []string{
							kubeNamespaceWeb,
							kubeStatefulsetWebDb,
							kubeSystemUid9833e04fA68947f9B58836a616432abd,
						},
						DataConfigMap: corev1.LocalObjectReference{Name: ihpaSample2Cpu},
						Provider: ihpav1beta2.MetricProvider{
							Name: sampleProvider,
							ProviderSource: ihpav1beta2.ProviderSource{
								Datadog: &ihpav1beta2.DatadogProviderSource{
									APIKey: xxxValue,
									APPKey: yyyValue,
								},
							},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      ihpaSample2Memory,
						Namespace: webValue3,
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion:         ihpaAkeCyberagentCoJpV1Value,
								Controller:         func(b bool) *bool { return &b }(true),
								BlockOwnerDeletion: func(b bool) *bool { return &b }(true),
								Kind:               intelligentHorizontalPodAutoscalerIhpaAkeCyberagentCoJpValue2,
								Name:               sample2Value3,
								UID:                id9fc642f3Bb9d404dAfd5D04f1d6149adValue,
							},
						},
						Annotations: map[string]string{
							ihpaAkeCyberagentCoJpFittingjobId: "c5abea3be857f41dc43ed7dec2eee7e3",
						},
					},
					Spec: ihpav1beta2.EstimatorSpec{
						Mode:       rawValue,
						GapMinutes: 10,
						MetricName: "ake.ihpa.forecasted_kubernetes_memory_usage",
						MetricTags: []string{
							kubeNamespaceWeb,
							kubeStatefulsetWebDb,
							kubeSystemUid9833e04fA68947f9B58836a616432abd,
						},
						BaseMetricName: "kubernetes.memory.usage",
						BaseMetricTags: []string{
							kubeNamespaceWeb,
							kubeStatefulsetWebDb,
							kubeSystemUid9833e04fA68947f9B58836a616432abd,
						},
						DataConfigMap: corev1.LocalObjectReference{Name: ihpaSample2Memory},
						Provider: ihpav1beta2.MetricProvider{
							Name: sampleProvider,
							ProviderSource: ihpav1beta2.ProviderSource{
								Datadog: &ihpav1beta2.DatadogProviderSource{
									APIKey: xxxValue,
									APPKey: yyyValue,
								},
							},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      ihpaSample2NginxNetRequestPerS,
						Namespace: webValue3,
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion:         ihpaAkeCyberagentCoJpV1Value,
								Controller:         func(b bool) *bool { return &b }(true),
								BlockOwnerDeletion: func(b bool) *bool { return &b }(true),
								Kind:               intelligentHorizontalPodAutoscalerIhpaAkeCyberagentCoJpValue2,
								Name:               sample2Value3,
								UID:                id9fc642f3Bb9d404dAfd5D04f1d6149adValue,
							},
						},
						Annotations: map[string]string{
							ihpaAkeCyberagentCoJpFittingjobId: "e9932760a233dffd256e6e0de941096e",
						},
					},
					Spec: ihpav1beta2.EstimatorSpec{
						Mode:       rawValue,
						GapMinutes: 10,
						MetricName: "ake.ihpa.forecasted_nginx_net_request_per_s",
						MetricTags: []string{
							kubeNamespaceWeb,
							kubeStatefulsetWebDb,
							kubeSystemUid9833e04fA68947f9B58836a616432abd,
						},
						BaseMetricName: nginxNetRequestPerSValue2,
						BaseMetricTags: []string{
							"kube_app:sample",
							kubeNamespaceWeb,
							kubeStatefulsetWebDb,
						},
						DataConfigMap: corev1.LocalObjectReference{Name: ihpaSample2NginxNetRequestPerS},
						Provider: ihpav1beta2.MetricProvider{
							Name: sampleProvider,
							ProviderSource: ihpav1beta2.ProviderSource{
								Datadog: &ihpav1beta2.DatadogProviderSource{
									APIKey: xxxValue,
									APPKey: yyyValue,
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
	const webValue2 = "web"
	const sample2Value2 = "sample2"
	const sample1Value2 = "sample1"
	const ihpaAkeCyberagentCoJpV1beta1 = "ihpa.ake.cyberagent.co.jp/v1beta1"
	const ihpaAkeCyberagentCoJpV1 = "ihpa.ake.cyberagent.co.jp/v1"
	const ihpaSample2 = "ihpa-sample2"
	const ihpaSample1 = "ihpa-sample1"
	const defaultValue4 = "default"
	const intelligentHorizontalPodAutoscalerIhpaAkeCyberagentCoJpValue = "IntelligentHorizontalPodAutoscaler.ihpa.ake.cyberagent.co.jp"
	const id9fc642f3Bb9d404dAfd5D04f1d6149ad = "9fc642f3-bb9d-404d-afd5-d04f1d6149ad"
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
					Name:      ihpaSample1,
					Namespace: defaultValue4,
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         ihpaAkeCyberagentCoJpV1beta1,
							Controller:         func(b bool) *bool { return &b }(true),
							BlockOwnerDeletion: func(b bool) *bool { return &b }(true),
							Kind:               intelligentHorizontalPodAutoscalerIhpaAkeCyberagentCoJpValue,
							Name:               sample1Value2,
							UID:                id9fc642f3Bb9d404dAfd5D04f1d6149ad,
						},
					},
				},
			},
			expectedRole: &rbacv1.Role{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ihpaSample1,
					Namespace: defaultValue4,
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         ihpaAkeCyberagentCoJpV1beta1,
							Controller:         func(b bool) *bool { return &b }(true),
							BlockOwnerDeletion: func(b bool) *bool { return &b }(true),
							Kind:               intelligentHorizontalPodAutoscalerIhpaAkeCyberagentCoJpValue,
							Name:               sample1Value2,
							UID:                id9fc642f3Bb9d404dAfd5D04f1d6149ad,
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
					Name:      ihpaSample1,
					Namespace: defaultValue4,
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         ihpaAkeCyberagentCoJpV1beta1,
							Controller:         func(b bool) *bool { return &b }(true),
							BlockOwnerDeletion: func(b bool) *bool { return &b }(true),
							Kind:               intelligentHorizontalPodAutoscalerIhpaAkeCyberagentCoJpValue,
							Name:               sample1Value2,
							UID:                id9fc642f3Bb9d404dAfd5D04f1d6149ad,
						},
					},
				},
				RoleRef: rbacv1.RoleRef{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "Role",
					Name:     ihpaSample1,
				},
				Subjects: []rbacv1.Subject{
					{
						APIGroup:  "",
						Kind:      "ServiceAccount",
						Name:      ihpaSample1,
						Namespace: defaultValue4,
					},
				},
			},
		},
		{
			generator: sample2,
			expectedSA: &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ihpaSample2,
					Namespace: webValue2,
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         ihpaAkeCyberagentCoJpV1,
							Controller:         func(b bool) *bool { return &b }(true),
							BlockOwnerDeletion: func(b bool) *bool { return &b }(true),
							Kind:               intelligentHorizontalPodAutoscalerIhpaAkeCyberagentCoJpValue,
							Name:               sample2Value2,
							UID:                id9fc642f3Bb9d404dAfd5D04f1d6149ad,
						},
					},
				},
			},
			expectedRole: &rbacv1.Role{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ihpaSample2,
					Namespace: webValue2,
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         ihpaAkeCyberagentCoJpV1,
							Controller:         func(b bool) *bool { return &b }(true),
							BlockOwnerDeletion: func(b bool) *bool { return &b }(true),
							Kind:               intelligentHorizontalPodAutoscalerIhpaAkeCyberagentCoJpValue,
							Name:               sample2Value2,
							UID:                id9fc642f3Bb9d404dAfd5D04f1d6149ad,
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
					Name:      ihpaSample2,
					Namespace: webValue2,
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         ihpaAkeCyberagentCoJpV1,
							Controller:         func(b bool) *bool { return &b }(true),
							BlockOwnerDeletion: func(b bool) *bool { return &b }(true),
							Kind:               intelligentHorizontalPodAutoscalerIhpaAkeCyberagentCoJpValue,
							Name:               sample2Value2,
							UID:                id9fc642f3Bb9d404dAfd5D04f1d6149ad,
						},
					},
				},
				RoleRef: rbacv1.RoleRef{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "Role",
					Name:     ihpaSample2,
				},
				Subjects: []rbacv1.Subject{
					{
						APIGroup:  "",
						Kind:      "ServiceAccount",
						Name:      ihpaSample2,
						Namespace: webValue2,
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

func TestStatusConfigMapResource(t *testing.T) {
	const webValue = "web"
	const sample2Value = "sample2"
	const sample1Value = "sample1"
	const ihpaSample1Cpu = "ihpa-sample1-cpu"
	const defaultValue3 = "default"
	const intelligentHorizontalPodAutoscalerIhpaAkeCyberagentCoJp = "IntelligentHorizontalPodAutoscaler.ihpa.ake.cyberagent.co.jp"
	sample1, sample2 := testIHPAGeneratorSample(t)
	tests := []struct {
		name      string
		generator *ihpaGeneratorImpl
		expected  *corev1.ConfigMap
	}{
		{
			name:      sample1Value,
			generator: sample1,
			expected: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ihpa-sample1-status",
					Namespace: defaultValue3,
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         "ihpa.ake.cyberagent.co.jp/v1beta1",
							Controller:         func(b bool) *bool { return &b }(true),
							BlockOwnerDeletion: func(b bool) *bool { return &b }(true),
							Kind:               intelligentHorizontalPodAutoscalerIhpaAkeCyberagentCoJp,
							Name:               sample1Value,
							UID:                "9fc642f3-bb9d-404d-afd5-d04f1d6149ad",
						},
					},
				},
				Data: map[string]string{
					"hpaName":         "ihpa-sample1",
					"fittingJobNames": ihpaSample1Cpu,
					"estimatorNames":  ihpaSample1Cpu,
				},
			},
		},
		{
			name:      sample2Value,
			generator: sample2,
			expected: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ihpa-sample2-status",
					Namespace: webValue,
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         "ihpa.ake.cyberagent.co.jp/v1",
							Controller:         func(b bool) *bool { return &b }(true),
							BlockOwnerDeletion: func(b bool) *bool { return &b }(true),
							Kind:               intelligentHorizontalPodAutoscalerIhpaAkeCyberagentCoJp,
							Name:               sample2Value,
							UID:                "9fc642f3-bb9d-404d-afd5-d04f1d6149ad",
						},
					},
				},
				Data: map[string]string{
					"hpaName":         "ihpa-sample2",
					"fittingJobNames": "ihpa-sample2-cpu,ihpa-sample2-memory,ihpa-sample2-nginx-net-request-per-s",
					"estimatorNames":  "ihpa-sample2-cpu,ihpa-sample2-memory,ihpa-sample2-nginx-net-request-per-s",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.generator.StatusConfigMapResource()
			if err != nil {
				t.Fatal(err)
			}
			if got.Name != tt.expected.Name {
				t.Errorf("configmap name mismatch: got=%s, exp=%s", got.Name, tt.expected.Name)
			}
			if got.Namespace != tt.expected.Namespace {
				t.Errorf("configmap namespace mismatch: got=%s, exp=%s", got.Namespace, tt.expected.Namespace)
			}
			if !reflect.DeepEqual(got.Data, tt.expected.Data) {
				t.Errorf("configmap data mismatch:\ngot=%#v\nexp=%#v", got.Data, tt.expected.Data)
			}
			if len(got.OwnerReferences) != 1 {
				t.Fatalf("expected 1 owner reference, got %d", len(got.OwnerReferences))
			}
			if got.OwnerReferences[0].Name != tt.expected.OwnerReferences[0].Name {
				t.Errorf("owner reference name mismatch: got=%s, exp=%s", got.OwnerReferences[0].Name, tt.expected.OwnerReferences[0].Name)
			}
		})
	}
}

func TestGenerateForecastedMetricSpec(t *testing.T) {
	const nginxNetRequestPerSValue = "nginx.net.request_per_s"
	const nginxValue = "nginx"
	const kubeSystemUidValue = "kube_system_uid"
	const kubeStatefulset = "kube_statefulset"
	const kubeNamespaceValue = "kube_namespace"
	const kubeDeploymentValue = "kube_deployment"
	const defaultValue2 = "default"
	const cpuValue3 = "cpu"
	const utilizationValue2 = "Utilization"
	const resourceValue3 = "Resource"
	const externalValue2 = "External"
	const averageValueValue3 = "AverageValue"
	const id46f9e396D3c44103A80749054f47bbfb = "46f9e396-d3c4-4103-a807-49054f47bbfb"
	sample1, sample2 := testIHPAGeneratorSample(t)
	tests := []struct {
		generator *ihpaGeneratorImpl
		metric    *autoscalingv2.MetricSpec
		expected  *autoscalingv2.MetricSpec
	}{
		{
			generator: sample1,
			metric: &autoscalingv2.MetricSpec{
				Type: resourceValue3,
				Resource: &autoscalingv2.ResourceMetricSource{
					Name: cpuValue3,
					Target: autoscalingv2.MetricTarget{
						Type:               utilizationValue2,
						AverageUtilization: func(i int32) *int32 { return &i }(30),
					},
				},
			},
			expected: &autoscalingv2.MetricSpec{
				Type: externalValue2,
				External: &autoscalingv2.ExternalMetricSource{
					Metric: autoscalingv2.MetricIdentifier{
						Name: "ake.ihpa.forecasted_kubernetes_cpu_usage_total",
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								kubeSystemUidValue:  id46f9e396D3c44103A80749054f47bbfb,
								kubeNamespaceValue:  defaultValue2,
								kubeDeploymentValue: nginxValue,
							},
						},
					},
					Target: autoscalingv2.MetricTarget{
						Type:         averageValueValue3,
						AverageValue: resource.NewQuantity(resource.NewQuantity(150, resource.DecimalSI).ScaledValue(resource.Micro), resource.DecimalSI),
					},
				},
			},
		},
		{
			generator: sample2,
			metric: &autoscalingv2.MetricSpec{
				Type: externalValue2,
				External: &autoscalingv2.ExternalMetricSource{
					Metric: autoscalingv2.MetricIdentifier{
						Name: nginxNetRequestPerSValue,
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"ownlabel": "hello",
							},
						},
					},
					Target: autoscalingv2.MetricTarget{
						Type:         averageValueValue3,
						AverageValue: resource.NewQuantity(100, resource.DecimalSI),
					},
				},
			},
			expected: &autoscalingv2.MetricSpec{
				Type: externalValue2,
				External: &autoscalingv2.ExternalMetricSource{
					Metric: autoscalingv2.MetricIdentifier{
						Name: "ake.ihpa.forecasted_nginx_net_request_per_s",
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								kubeSystemUidValue: "9833e04f-a689-47f9-b588-36a616432abd",
								kubeNamespaceValue: "web",
								kubeStatefulset:    "web-db",
							},
						},
					},
					Target: autoscalingv2.MetricTarget{
						Type:         averageValueValue3,
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
	const requestsPerSecond = "requests-per-second"
	const packetsPerSecond = "packets-per-second"
	const nginxNetRequestPerS = "nginx.net.request_per_s"
	const kubeDeployment = "kube_deployment"
	const cpuValue2 = "cpu"
	const resourceValue2 = "Resource"
	const externalValue = "External"
	const averageValueValue2 = "AverageValue"
	sample1, sample2 := testIHPAGeneratorSample(t)
	tests := []struct {
		generator *ihpaGeneratorImpl
		metric    *autoscalingv2.MetricSpec
		expected  *autoscalingv2.MetricIdentifier
	}{
		{
			generator: sample1,
			metric: &autoscalingv2.MetricSpec{
				Type: resourceValue2,
				Resource: &autoscalingv2.ResourceMetricSource{
					Name: cpuValue2,
					Target: autoscalingv2.MetricTarget{
						Type:               "Utilization",
						AverageUtilization: func(i int32) *int32 { return &i }(30),
					},
				},
			},
			expected: &autoscalingv2.MetricIdentifier{
				Name: "kubernetes.cpu.usage.total",
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"kube_system_uid": "46f9e396-d3c4-4103-a807-49054f47bbfb",
						"kube_namespace":  "default",
						kubeDeployment:    "nginx",
					},
				},
			},
		},
		{
			generator: sample2,
			metric: &autoscalingv2.MetricSpec{
				Type: externalValue,
				External: &autoscalingv2.ExternalMetricSource{
					Metric: autoscalingv2.MetricIdentifier{
						Name: nginxNetRequestPerS,
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"super_identifier": "hi",
							},
						},
					},
					Target: autoscalingv2.MetricTarget{
						Type:         averageValueValue2,
						AverageValue: resource.NewQuantity(100, resource.DecimalSI),
					},
				},
			},
			expected: &autoscalingv2.MetricIdentifier{
				Name: nginxNetRequestPerS,
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"super_identifier": "hi",
					},
				},
			},
		},
		{
			generator: sample2,
			metric: &autoscalingv2.MetricSpec{
				Type: "Object",
				Object: &autoscalingv2.ObjectMetricSource{
					Metric: autoscalingv2.MetricIdentifier{
						Name: requestsPerSecond,
					},
					DescribedObject: autoscalingv2.CrossVersionObjectReference{
						Kind:       "Ingress",
						Name:       "main-route",
						APIVersion: "networking.k8s.io/v1beta1",
					},
					Target: autoscalingv2.MetricTarget{
						Type:  "Value",
						Value: resource.NewQuantity(2000, resource.DecimalSI),
					},
				},
			},
			expected: &autoscalingv2.MetricIdentifier{
				Name: requestsPerSecond,
			},
		},
		{
			generator: sample2,
			metric: &autoscalingv2.MetricSpec{
				Type: "Pods",
				Pods: &autoscalingv2.PodsMetricSource{
					Metric: autoscalingv2.MetricIdentifier{
						Name: packetsPerSecond,
					},
					Target: autoscalingv2.MetricTarget{
						Type:         averageValueValue2,
						AverageValue: resource.NewQuantity(100, resource.DecimalSI),
					},
				},
			},
			expected: &autoscalingv2.MetricIdentifier{
				Name: packetsPerSecond,
			},
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
	const utilizationValue = "Utilization"
	const resourceValue = "Resource"
	const averageValueValue = "AverageValue"
	sample1, sample2 := testIHPAGeneratorSample(t)
	tests := []struct {
		generator *ihpaGeneratorImpl
		metric    *autoscalingv2.MetricSpec
		expected  string
	}{
		{
			generator: sample1,
			metric: &autoscalingv2.MetricSpec{
				Type: resourceValue,
				Resource: &autoscalingv2.ResourceMetricSource{
					Name: "cpu",
					Target: autoscalingv2.MetricTarget{
						Type:               utilizationValue,
						AverageUtilization: func(i int32) *int32 { return &i }(30),
					},
				},
			},
			expected: "b9aa658a6b30a452ee08d0fbea2c4f40",
		},
		{
			generator: sample1,
			metric: &autoscalingv2.MetricSpec{
				Type: "External",
				External: &autoscalingv2.ExternalMetricSource{
					Metric: autoscalingv2.MetricIdentifier{
						Name: "hello_world.metric",
					},
					Target: autoscalingv2.MetricTarget{
						Type:         averageValueValue,
						AverageValue: resource.NewQuantity(100, resource.DecimalSI),
					},
				},
			},
			expected: "3f4d9eb4d50045a484a41fb3ba466dd6",
		},
		{
			generator: sample2,
			metric: &autoscalingv2.MetricSpec{
				Type: resourceValue,
				Resource: &autoscalingv2.ResourceMetricSource{
					Name: "memory",
					Target: autoscalingv2.MetricTarget{
						Type:               utilizationValue,
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
	const kubeSystemUid = "kube_system_uid"
	const kubeNamespace = "kube_namespace"
	sample1, sample2 := testIHPAGeneratorSample(t)
	tests := []struct {
		generator *ihpaGeneratorImpl
		expected  map[string]string
	}{
		{
			generator: sample1,
			expected: map[string]string{
				kubeSystemUid:     "46f9e396-d3c4-4103-a807-49054f47bbfb",
				kubeNamespace:     "default",
				"kube_deployment": "nginx",
			},
		},
		{
			generator: sample2,
			expected: map[string]string{
				kubeSystemUid:      "9833e04f-a689-47f9-b588-36a616432abd",
				kubeNamespace:      "web",
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
	const cpuValue = "cpu"
	const utilization = "Utilization"
	const external = "External"
	const averageValue = "AverageValue"
	sample1, sample2 := testIHPAGeneratorSample(t)
	tests := []struct {
		generator *ihpaGeneratorImpl
		metric    *autoscalingv2.MetricSpec
		expected  string
	}{
		{
			generator: sample1,
			metric: &autoscalingv2.MetricSpec{
				Type: "Resource",
				Resource: &autoscalingv2.ResourceMetricSource{
					Name: cpuValue,
					Target: autoscalingv2.MetricTarget{
						Type:               utilization,
						AverageUtilization: func(i int32) *int32 { return &i }(30),
					},
				},
			},
			expected: "ihpa-sample1-cpu",
		},
		{
			generator: sample1,
			metric: &autoscalingv2.MetricSpec{
				Type: external,
				Resource: &autoscalingv2.ResourceMetricSource{
					Name: cpuValue,
					Target: autoscalingv2.MetricTarget{
						Type:               utilization,
						AverageUtilization: func(i int32) *int32 { return &i }(30),
					},
				},
				External: &autoscalingv2.ExternalMetricSource{
					Metric: autoscalingv2.MetricIdentifier{
						Name: "hello_world.metric",
					},
					Target: autoscalingv2.MetricTarget{
						Type:         averageValue,
						AverageValue: resource.NewQuantity(100, resource.DecimalSI),
					},
				},
			},
			expected: "ihpa-sample1-hello-world-metric",
		},
		{
			generator: sample2,
			metric: &autoscalingv2.MetricSpec{
				Type: external,
				External: &autoscalingv2.ExternalMetricSource{
					Metric: autoscalingv2.MetricIdentifier{
						Name: "nginx.net.request_per_s",
					},
					Target: autoscalingv2.MetricTarget{
						Type:         averageValue,
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

func TestResolveDatadogKeysFromEnv(t *testing.T) {
	const defaultValue = "default"
	const aPPKey = "APPKey"
	const aPIKey = "APIKey"
	tests := []struct {
		name        string
		namespace   string
		datadog     *ihpav1beta2.DatadogProviderSource
		secrets     []runtime.Object
		configMaps  []runtime.Object
		expectAPI   string
		expectAPP   string
		expectError bool
	}{
		{
			name:      "resolve keys from secret",
			namespace: defaultValue,
			datadog: &ihpav1beta2.DatadogProviderSource{
				KeysFrom: []corev1.EnvFromSource{
					{
						SecretRef: &corev1.SecretEnvSource{
							LocalObjectReference: corev1.LocalObjectReference{Name: "datadog-keys"},
						},
					},
				},
			},
			secrets: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{Name: "datadog-keys", Namespace: defaultValue},
					Data: map[string][]byte{
						aPIKey: []byte("secret-api-key"),
						aPPKey: []byte("secret-app-key"),
					},
				},
			},
			expectAPI: "secret-api-key",
			expectAPP: "secret-app-key",
		},
		{
			name:      "resolve keys from configmap",
			namespace: defaultValue,
			datadog: &ihpav1beta2.DatadogProviderSource{
				KeysFrom: []corev1.EnvFromSource{
					{
						ConfigMapRef: &corev1.ConfigMapEnvSource{
							LocalObjectReference: corev1.LocalObjectReference{Name: "datadog-keys-cm"},
						},
					},
				},
			},
			configMaps: []runtime.Object{
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{Name: "datadog-keys-cm", Namespace: defaultValue},
					Data: map[string]string{
						aPIKey: "cm-api-key",
						aPPKey: "cm-app-key",
					},
				},
			},
			expectAPI: "cm-api-key",
			expectAPP: "cm-app-key",
		},
		{
			name:      "resolve keys from secret with prefix",
			namespace: defaultValue,
			datadog: &ihpav1beta2.DatadogProviderSource{
				KeysFrom: []corev1.EnvFromSource{
					{
						SecretRef: &corev1.SecretEnvSource{
							LocalObjectReference: corev1.LocalObjectReference{Name: "dd-keys"},
						},
					},
				},
			},
			secrets: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{Name: "dd-keys", Namespace: defaultValue},
					Data: map[string][]byte{
						aPIKey: []byte("api-from-secret"),
					},
				},
			},
			expectAPI: "api-from-secret",
			expectAPP: "",
		},
		{
			name:      "optional secret not found is skipped",
			namespace: defaultValue,
			datadog: &ihpav1beta2.DatadogProviderSource{
				KeysFrom: []corev1.EnvFromSource{
					{
						SecretRef: &corev1.SecretEnvSource{
							LocalObjectReference: corev1.LocalObjectReference{Name: "nonexistent"},
							Optional:             func(b bool) *bool { return &b }(true),
						},
					},
				},
			},
			expectAPI: "",
			expectAPP: "",
		},
		{
			name:      "required secret not found returns error",
			namespace: defaultValue,
			datadog: &ihpav1beta2.DatadogProviderSource{
				KeysFrom: []corev1.EnvFromSource{
					{
						SecretRef: &corev1.SecretEnvSource{
							LocalObjectReference: corev1.LocalObjectReference{Name: "nonexistent"},
						},
					},
				},
			},
			expectError: true,
		},
		{
			name:      "multiple sources - secret overrides configmap",
			namespace: defaultValue,
			datadog: &ihpav1beta2.DatadogProviderSource{
				KeysFrom: []corev1.EnvFromSource{
					{
						ConfigMapRef: &corev1.ConfigMapEnvSource{
							LocalObjectReference: corev1.LocalObjectReference{Name: "cm-keys"},
						},
					},
					{
						SecretRef: &corev1.SecretEnvSource{
							LocalObjectReference: corev1.LocalObjectReference{Name: "secret-keys"},
						},
					},
				},
			},
			secrets: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{Name: "secret-keys", Namespace: defaultValue},
					Data: map[string][]byte{
						aPIKey: []byte("final-api-key"),
						aPPKey: []byte("final-app-key"),
					},
				},
			},
			configMaps: []runtime.Object{
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{Name: "cm-keys", Namespace: defaultValue},
					Data: map[string]string{
						aPIKey: "initial-api-key",
						aPPKey: "initial-app-key",
					},
				},
			},
			expectAPI: "final-api-key",
			expectAPP: "final-app-key",
		},
		{
			name:      "no keysFrom - no change",
			namespace: defaultValue,
			datadog: &ihpav1beta2.DatadogProviderSource{
				APIKey: "original-api",
				APPKey: "original-app",
			},
			expectAPI: "original-api",
			expectAPP: "original-app",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			objs := append(tt.secrets, tt.configMaps...)
			c := fake.NewFakeClient(objs...)
			err := resolveDatadogKeysFromEnv(context.Background(), c, tt.namespace, tt.datadog)
			if tt.expectError {
				if err == nil {
					t.Fatal("expected error but got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.datadog.APIKey != tt.expectAPI {
				t.Fatalf("APIKey mismatch: got=%s, exp=%s", tt.datadog.APIKey, tt.expectAPI)
			}
			if tt.datadog.APPKey != tt.expectAPP {
				t.Fatalf("APPKey mismatch: got=%s, exp=%s", tt.datadog.APPKey, tt.expectAPP)
			}
		})
	}
}
